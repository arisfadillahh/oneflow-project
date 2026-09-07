package httpapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"oneflow/wa-gateway/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lib/pq"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

type realTransport struct {
	cfg        config.Config
	db         *pgxpool.Pool
	httpClient *http.Client

	mu        sync.RWMutex
	container *sqlstore.Container
	clients   map[string]*waSessionRuntime
	legacy    *waSessionRuntime
}

type waSessionRuntime struct {
	sessionID     string
	displayName   string
	client        *whatsmeow.Client
	qrCode        string
	qrExpires     time.Time
	lastStatus    string
	lastInfo      map[string]any
	connecting    bool
	qrChannelOpen bool
	qrCancel      context.CancelFunc
}

type inboundImagePayload struct {
	Base64   string
	MimeType string
	Caption  string
	Size     int
}

func newRealTransport(cfg config.Config, db *pgxpool.Pool, httpClient *http.Client) transport {
	rt := &realTransport{
		cfg:        cfg,
		db:         db,
		httpClient: httpClient,
		clients:    map[string]*waSessionRuntime{},
	}
	if err := rt.init(context.Background()); err != nil {
		_ = upsertStatus(context.Background(), db, "degraded", map[string]any{
			"mode":      "real",
			"needsQr":   true,
			"lastError": err.Error(),
		})
	}
	return rt
}

func (rt *realTransport) init(ctx context.Context) error {
	sqlstore.PostgresArrayWrapper = pq.Array
	container, err := sqlstore.New(ctx, rt.cfg.SessionDialect, rt.cfg.SessionDSN, waLog.Noop)
	if err != nil {
		return err
	}
	rt.mu.Lock()
	rt.container = container
	rt.legacy = &waSessionRuntime{
		sessionID:  "",
		lastStatus: "disconnected",
		lastInfo: map[string]any{
			"mode":    "real",
			"needsQr": true,
		},
	}
	rt.mu.Unlock()
	rt.reconnectConfiguredSessions(ctx)
	return nil
}

func (rt *realTransport) reconnectConfiguredSessions(ctx context.Context) {
	rows, err := rt.db.Query(ctx, `
		SELECT id::text
		FROM whatsapp_sessions
		WHERE status IN ('connected', 'connecting') AND deleted_at IS NULL
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var sessionID string
		if err := rows.Scan(&sessionID); err != nil || strings.TrimSpace(sessionID) == "" {
			continue
		}
		go func(id string) {
			connectCtx, cancel := context.WithTimeout(context.WithValue(context.Background(), contextWhatsAppSessionID, id), 30*time.Second)
			defer cancel()
			_ = rt.connect(connectCtx)
		}(sessionID)
	}
}

func (rt *realTransport) runtimeFor(ctx context.Context) (*waSessionRuntime, error) {
	sessionID := sessionIDFromContext(ctx)
	rt.mu.RLock()
	if sessionID == "" && rt.legacy != nil && rt.legacy.client != nil {
		runtime := rt.legacy
		rt.mu.RUnlock()
		return runtime, nil
	}
	if sessionID != "" && rt.clients[sessionID] != nil {
		runtime := rt.clients[sessionID]
		rt.mu.RUnlock()
		return runtime, nil
	}
	container := rt.container
	rt.mu.RUnlock()
	if container == nil {
		return nil, fmt.Errorf("whatsapp store is not initialized")
	}

	var runtime *waSessionRuntime
	var deviceStore *store.Device
	var err error
	if sessionID == "" {
		device, deviceErr := container.GetFirstDevice(ctx)
		if deviceErr != nil {
			return nil, deviceErr
		}
		deviceStore = device
		runtime = &waSessionRuntime{
			sessionID:   "",
			displayName: "Oneflow.id",
			lastStatus:  "disconnected",
			lastInfo: map[string]any{
				"mode":    "real",
				"needsQr": true,
			},
		}
	} else {
		var status string
		var label string
		var detailsRaw string
		err = rt.db.QueryRow(ctx, `
			SELECT status, label, COALESCE(details, '{}'::jsonb)::text
			FROM whatsapp_sessions
			WHERE id = $1 AND deleted_at IS NULL
		`, sessionID).Scan(&status, &label, &detailsRaw)
		if err != nil {
			return nil, err
		}
		details := map[string]any{}
		_ = json.Unmarshal([]byte(detailsRaw), &details)
		deviceJID := strings.TrimSpace(fmt.Sprint(details["deviceJid"]))
		if deviceJID == "" {
			deviceJID = strings.TrimSpace(fmt.Sprint(details["deviceJID"]))
		}
		if deviceJID != "" {
			jid, parseErr := types.ParseJID(deviceJID)
			if parseErr == nil {
				device, deviceErr := container.GetDevice(ctx, jid)
				if deviceErr != nil {
					return nil, deviceErr
				}
				if device != nil {
					deviceStore = device
				}
			}
		}
		if deviceStore == nil {
			deviceStore = container.NewDevice()
		}
		runtime = &waSessionRuntime{
			sessionID:   sessionID,
			displayName: whatsappDeviceDisplayName(label),
			lastStatus:  normalizeSessionStatus(status),
			lastInfo: map[string]any{
				"mode":      "real",
				"needsQr":   status != "connected",
				"sessionId": sessionID,
			},
		}
		for key, value := range details {
			runtime.lastInfo[key] = value
		}
	}

	if deviceStore == nil {
		return nil, fmt.Errorf("failed to initialize whatsapp device")
	}
	return rt.attachRuntime(ctx, sessionID, runtime, deviceStore)
}

func (rt *realTransport) attachRuntime(ctx context.Context, sessionID string, runtime *waSessionRuntime, device *store.Device) (*waSessionRuntime, error) {
	store.SetOSInfo(runtime.displayName, [3]uint32{1, 0, 0})
	client := whatsmeow.NewClient(device, waLog.Noop)
	client.AddEventHandler(func(raw interface{}) {
		rt.handleEventFor(sessionID, raw)
	})
	runtime.client = client
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if sessionID == "" {
		rt.legacy = runtime
		return rt.legacy, nil
	}
	if existing := rt.clients[sessionID]; existing != nil {
		if client != nil {
			client.Disconnect()
		}
		return existing, nil
	}
	rt.clients[sessionID] = runtime
	_ = ctx
	return runtime, nil
}

func (rt *realTransport) Status(ctx context.Context) (map[string]any, error) {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return nil, err
	}
	rt.mu.RLock()
	status := runtime.lastStatus
	details := runtime.lastInfo
	rt.mu.RUnlock()
	if status == "connected" && !rt.hasUsableSession(ctx) {
		details = map[string]any{
			"mode":      "real",
			"mock":      false,
			"needsQr":   true,
			"lastError": "connected_event_without_valid_session",
			"checkedAt": time.Now().UTC(),
		}
		_ = rt.persistStatus(ctx, "disconnected", details)
		status = "disconnected"
	}
	return map[string]any{
		"status":  status,
		"details": details,
	}, nil
}

func (rt *realTransport) GetQR(ctx context.Context) (map[string]any, error) {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return nil, err
	}
	rt.mu.RLock()
	if runtime.qrCode != "" && time.Now().Before(runtime.qrExpires) {
		code := runtime.qrCode
		expiresIn := int(time.Until(runtime.qrExpires).Seconds())
		rt.mu.RUnlock()
		return map[string]any{
			"status":    "qr_ready",
			"mode":      "real",
			"mock":      false,
			"qr":        code,
			"expiresIn": expiresIn,
		}, nil
	}
	rt.mu.RUnlock()

	if _, err := rt.Connect(ctx); err != nil {
		rt.mu.RLock()
		defer rt.mu.RUnlock()
		if runtime.qrCode != "" && time.Now().Before(runtime.qrExpires) {
			return map[string]any{
				"status":    "qr_ready",
				"mode":      "real",
				"mock":      false,
				"qr":        runtime.qrCode,
				"expiresIn": int(time.Until(runtime.qrExpires).Seconds()),
				"warning":   err.Error(),
			}, nil
		}
		return nil, err
	}

	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return map[string]any{
		"status":    "qr_pending",
		"mode":      "real",
		"mock":      false,
		"qr":        runtime.qrCode,
		"expiresIn": int(time.Until(runtime.qrExpires).Seconds()),
	}, nil
}

func (rt *realTransport) Connect(ctx context.Context) (map[string]any, error) {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return nil, err
	}
	if err := rt.connect(ctx); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		rt.mu.RLock()
		qrReady := runtime.qrCode != "" && time.Now().Before(runtime.qrExpires)
		status := runtime.lastStatus
		rt.mu.RUnlock()
		if qrReady || status == "connected" {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(150 * time.Millisecond):
		}
	}
	return rt.Status(ctx)
}

func (rt *realTransport) PairPhone(ctx context.Context, phone string) (map[string]any, error) {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return nil, err
	}
	if runtime.client == nil {
		return nil, fmt.Errorf("whatsapp client not initialized")
	}
	if err := rt.connect(ctx); err != nil {
		return nil, err
	}
	code, err := runtime.client.PairPhone(ctx, normalizePhone(phone), false, whatsmeow.PairClientChrome, "Chrome (Windows)")
	if err != nil {
		return nil, err
	}
	_ = rt.persistStatus(ctx, "disconnected", map[string]any{
		"mode":      "real",
		"mock":      false,
		"needsQr":   false,
		"pairPhone": normalizePhone(phone),
		"pairCode":  code,
	})
	return map[string]any{
		"status":   "pair_code_ready",
		"mode":     "real",
		"mock":     false,
		"phone":    normalizePhone(phone),
		"pairCode": code,
		"hint":     "Enter this code in WhatsApp > Linked devices > Link with phone number instead.",
	}, nil
}

func (rt *realTransport) connect(ctx context.Context) error {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return err
	}
	rt.mu.RLock()
	client := runtime.client
	deviceIDKnown := client != nil && client.Store != nil && client.Store.ID != nil
	qrStillValid := runtime.qrCode != "" && time.Now().Before(runtime.qrExpires)
	qrChannelOpen := runtime.qrChannelOpen
	connecting := runtime.connecting
	rt.mu.RUnlock()
	if client == nil {
		return fmt.Errorf("whatsapp client not initialized")
	}
	if qrStillValid {
		return nil
	}
	if connecting {
		return nil
	}

	if !deviceIDKnown && !qrChannelOpen {
		qrCtx, qrCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		qrChan, err := client.GetQRChannel(qrCtx)
		if err != nil {
			qrCancel()
			return err
		}
		rt.mu.Lock()
		runtime.qrChannelOpen = true
		if runtime.qrCancel != nil {
			runtime.qrCancel()
		}
		runtime.qrCancel = qrCancel
		rt.mu.Unlock()
		go rt.consumeQRChannel(ctx, runtime, qrChan)
	}
	rt.mu.Lock()
	runtime.connecting = true
	rt.mu.Unlock()
	err = client.Connect()
	rt.mu.Lock()
	runtime.connecting = false
	rt.mu.Unlock()
	return err
}

func (rt *realTransport) Disconnect(ctx context.Context) (map[string]any, error) {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return nil, err
	}
	if runtime.client != nil {
		runtime.client.Disconnect()
	}
	rt.mu.Lock()
	if runtime.qrCancel != nil {
		runtime.qrCancel()
		runtime.qrCancel = nil
	}
	runtime.qrChannelOpen = false
	rt.mu.Unlock()
	payload := map[string]any{
		"mode":           "real",
		"mock":           false,
		"needsQr":        true,
		"disconnectedAt": time.Now().UTC(),
	}
	if err := rt.persistStatus(ctx, "disconnected", payload); err != nil {
		return nil, err
	}
	return map[string]any{"status": "disconnected", "details": payload}, nil
}

func (rt *realTransport) SendText(ctx context.Context, phone, text string) (map[string]any, error) {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return nil, err
	}
	rt.mu.RLock()
	client := runtime.client
	status := runtime.lastStatus
	rt.mu.RUnlock()
	if client == nil || status != "connected" || !rt.hasUsableSession(ctx) {
		return nil, fmt.Errorf("whatsapp real transport is not connected")
	}

	jid := recipientJID(phone)
	msg := &waProto.Message{Conversation: proto.String(text)}
	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status":            "sent",
		"mode":              "real",
		"externalMessageId": resp.ID,
		"phone":             phone,
	}, nil
}

func (rt *realTransport) SendMedia(ctx context.Context, media outboundMediaRequest) (map[string]any, error) {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return nil, err
	}
	rt.mu.RLock()
	client := runtime.client
	status := runtime.lastStatus
	rt.mu.RUnlock()
	if client == nil || status != "connected" || !rt.hasUsableSession(ctx) {
		return nil, fmt.Errorf("whatsapp real transport is not connected")
	}

	data, err := decodeBase64Payload(media.MediaBase64)
	if err != nil {
		return nil, fmt.Errorf("invalid media payload: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("media payload is empty")
	}
	if len(data) > 8*1024*1024 {
		return nil, fmt.Errorf("media payload is too large")
	}

	jid := recipientJID(media.Phone)
	mimeType := strings.TrimSpace(media.MediaMime)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	mediaKind := normalizeOutboundMediaKind(media.MediaKind, mimeType)
	uploadType := whatsmeow.MediaDocument
	if mediaKind == "image" {
		uploadType = whatsmeow.MediaImage
	} else if mediaKind == "audio" {
		uploadType = whatsmeow.MediaAudio
	}
	uploaded, err := client.Upload(ctx, data, uploadType)
	if err != nil {
		return nil, err
	}

	caption := strings.TrimSpace(media.Text)
	filename := strings.TrimSpace(media.MediaName)
	if filename == "" {
		filename = defaultOutboundMediaName(mimeType, mediaKind)
	}
	msg := &waProto.Message{}
	if mediaKind == "image" {
		msg.ImageMessage = &waProto.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
		}
	} else {
		msg.DocumentMessage = &waProto.DocumentMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			FileName:      proto.String(filename),
			Title:         proto.String(filename),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
		}
	}
	resp, err := client.SendMessage(ctx, jid, msg)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status":            "sent",
		"mode":              "real",
		"externalMessageId": resp.ID,
		"phone":             media.Phone,
		"mediaKind":         mediaKind,
	}, nil
}

func (rt *realTransport) SetChatPresence(ctx context.Context, phone string, active bool) (map[string]any, error) {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return nil, err
	}
	rt.mu.RLock()
	client := runtime.client
	status := runtime.lastStatus
	rt.mu.RUnlock()
	if client == nil || status != "connected" || !rt.hasUsableSession(ctx) {
		return nil, fmt.Errorf("whatsapp real transport is not connected")
	}

	state := types.ChatPresencePaused
	if active {
		state = types.ChatPresenceComposing
	}
	jid := recipientJID(phone)
	if err := client.SendChatPresence(ctx, jid, state, types.ChatPresenceMediaText); err != nil {
		return nil, err
	}
	return map[string]any{
		"status": "ok",
		"mode":   "real",
		"phone":  phone,
		"state":  string(state),
	}, nil
}

func defaultOutboundMediaName(mimeType, mediaKind string) string {
	if mediaKind == "image" {
		if strings.Contains(mimeType, "png") {
			return "image.png"
		}
		if strings.Contains(mimeType, "webp") {
			return "image.webp"
		}
		return "image.jpg"
	}
	if strings.Contains(mimeType, "pdf") {
		return "document.pdf"
	}
	return "attachment"
}

func (rt *realTransport) consumeQRChannel(ctx context.Context, runtime *waSessionRuntime, qrChan <-chan whatsmeow.QRChannelItem) {
	defer func() {
		rt.mu.Lock()
		runtime.qrChannelOpen = false
		if runtime.qrCancel != nil {
			runtime.qrCancel()
			runtime.qrCancel = nil
		}
		rt.mu.Unlock()
	}()
	for item := range qrChan {
		fmt.Printf("[wa-gateway] qr event=%s\n", item.Event)
		switch item.Event {
		case "code":
			sessionID := strings.TrimSpace(runtime.sessionID)
			rt.mu.Lock()
			runtime.qrCode = item.Code
			runtime.qrExpires = time.Now().Add(60 * time.Second)
			rt.mu.Unlock()
			_ = rt.persistStatus(ctx, "qr_pending", map[string]any{
				"mode":      "real",
				"mock":      false,
				"needsQr":   true,
				"qr":        item.Code,
				"expiresIn": 60,
				"sessionId": sessionID,
			})
		case "success":
			rt.mu.Lock()
			runtime.qrCode = ""
			runtime.qrExpires = time.Time{}
			runtime.qrChannelOpen = false
			rt.mu.Unlock()
		default:
			rt.mu.Lock()
			runtime.qrCode = ""
			runtime.qrExpires = time.Time{}
			runtime.qrChannelOpen = false
			rt.mu.Unlock()
			_ = rt.persistStatus(ctx, "disconnected", map[string]any{
				"mode":      "real",
				"mock":      false,
				"needsQr":   true,
				"qrEvent":   item.Event,
				"lastError": item.Event,
			})
		}
	}
}

func (rt *realTransport) handleEventFor(sessionID string, raw interface{}) {
	ctx := context.Background()
	if sessionID != "" {
		ctx = context.WithValue(ctx, contextWhatsAppSessionID, sessionID)
	}
	runtime := rt.runtimeBySession(sessionID)
	switch evt := raw.(type) {
	case *events.Connected:
		fmt.Printf("[wa-gateway] session=%s event=connected\n", sessionID)
		if !runtimeHasUsableSession(runtime) {
			_ = rt.persistStatus(ctx, "disconnected", map[string]any{
				"mode":      "real",
				"mock":      false,
				"needsQr":   true,
				"lastError": "connected_event_without_valid_session",
				"checkedAt": time.Now().UTC(),
			})
			return
		}
		details := map[string]any{
			"mode":        "real",
			"mock":        false,
			"needsQr":     false,
			"connectedAt": time.Now().UTC(),
			"sessionId":   sessionID,
		}
		if runtime.client != nil && runtime.client.Store != nil && runtime.client.Store.ID != nil {
			details["deviceJid"] = runtime.client.Store.ID.String()
		}
		_ = rt.persistStatus(ctx, "connected", details)
	case *events.Disconnected:
		fmt.Printf("[wa-gateway] session=%s event=disconnected\n", sessionID)
		_ = rt.persistStatus(ctx, "disconnected", map[string]any{
			"mode":           "real",
			"mock":           false,
			"needsQr":        true,
			"disconnectedAt": time.Now().UTC(),
			"sessionId":      sessionID,
		})
	case *events.LoggedOut:
		fmt.Printf("[wa-gateway] session=%s event=logged_out reason=%s\n", sessionID, evt.Reason.String())
		_ = rt.persistStatus(ctx, "disconnected", map[string]any{
			"mode":           "real",
			"mock":           false,
			"needsQr":        true,
			"loggedOutAt":    time.Now().UTC(),
			"lastDisconnect": evt.Reason.String(),
			"sessionId":      sessionID,
		})
	case *events.PairError:
		fmt.Printf("[wa-gateway] session=%s event=pair_error err=%v\n", sessionID, evt.Error)
	case *events.ClientOutdated:
		fmt.Printf("[wa-gateway] session=%s event=client_outdated\n", sessionID)
	case *events.QRScannedWithoutMultidevice:
		fmt.Printf("[wa-gateway] session=%s event=qr_scanned_without_multidevice\n", sessionID)
	case *events.Message:
		if runtime != nil {
			rt.handleIncomingMessage(ctx, runtime, evt)
		}
	}
}

func (rt *realTransport) hasUsableSession(ctx context.Context) bool {
	runtime, err := rt.runtimeFor(ctx)
	if err != nil {
		return false
	}
	return runtimeHasUsableSession(runtime)
}

func runtimeHasUsableSession(runtime *waSessionRuntime) bool {
	if runtime == nil {
		return false
	}
	client := runtime.client
	if client == nil || client.Store == nil || client.Store.ID == nil {
		return false
	}
	return true
}

func whatsappDeviceDisplayName(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "Oneflow.id"
	}
	if strings.Contains(strings.ToLower(label), "oneflow.id") {
		return label
	}
	return label + " - Oneflow.id"
}

func (rt *realTransport) handleIncomingMessage(ctx context.Context, runtime *waSessionRuntime, evt *events.Message) {
	if evt.Info.IsFromMe {
		return
	}
	if isGroupJID(evt.Info.Chat.String()) {
		rt.handleIncomingGroupMessage(ctx, evt)
		return
	}
	if shouldIgnoreIncomingChat(evt.Info.Chat.String()) {
		return
	}
	text := extractMessageText(evt.Message)
	imagePayload := rt.extractImagePayload(ctx, runtime, evt.Message)
	if text == "" && imagePayload != nil && imagePayload.Caption != "" {
		text = imagePayload.Caption
	}
	if text == "" && imagePayload != nil {
		text = "Mohon bantu cek gambar/screenshot ini."
	}
	if text == "" {
		return
	}
	phone := evt.Info.Chat.User
	if phone == "" {
		phone = evt.Info.Sender.User
	}
	payload := map[string]any{
		"whatsappSessionId": sessionIDFromContext(ctx),
		"phone":             "+" + normalizePhone(phone),
		"customerName":      evt.Info.PushName,
		"text":              text,
		"externalMessageId": evt.Info.ID,
		"receivedAt":        evt.Info.Timestamp,
		"contentType":       "text",
	}
	if imagePayload != nil {
		payload["contentType"] = "image"
		payload["imageBase64"] = imagePayload.Base64
		payload["imageMimeType"] = imagePayload.MimeType
		payload["imageSizeBytes"] = imagePayload.Size
	}
	headers := map[string]string{
		"X-Internal-Token": rt.cfg.InternalGatewayToken,
	}
	if err := postJSON(context.Background(), rt.httpClient, rt.cfg.AppBackendBaseURL+"/api/internal/wa/inbound", headers, payload); err != nil {
		log.Printf("[wa-gateway] session=%s inbound forwarding failed: %v", sessionIDFromContext(ctx), err)
	}
}

func (rt *realTransport) handleIncomingGroupMessage(ctx context.Context, evt *events.Message) {
	text := extractMessageText(evt.Message)
	if text == "" {
		return
	}
	payload := map[string]any{
		"whatsappSessionId": sessionIDFromContext(ctx),
		"groupId":           evt.Info.Chat.String(),
		"groupName":         "",
		"senderJid":         evt.Info.Sender.String(),
		"text":              text,
		"externalMessageId": evt.Info.ID,
		"receivedAt":        evt.Info.Timestamp,
	}
	headers := map[string]string{
		"X-Internal-Token": rt.cfg.InternalGatewayToken,
	}
	_ = postJSON(context.Background(), rt.httpClient, rt.cfg.AppBackendBaseURL+"/api/internal/wa/group-binding", headers, payload)
}

func (rt *realTransport) extractImagePayload(ctx context.Context, runtime *waSessionRuntime, message *waProto.Message) *inboundImagePayload {
	if message == nil || message.GetImageMessage() == nil {
		return nil
	}
	image := message.GetImageMessage()
	rt.mu.RLock()
	client := runtime.client
	rt.mu.RUnlock()
	if client == nil {
		return nil
	}
	data, err := client.Download(ctx, image)
	if err != nil || len(data) == 0 {
		fmt.Printf("[wa-gateway] image_download_failed err=%v\n", err)
		return nil
	}
	if len(data) > 4*1024*1024 {
		fmt.Printf("[wa-gateway] image_download_skipped reason=too_large size=%d\n", len(data))
		return nil
	}
	mimeType := strings.TrimSpace(image.GetMimetype())
	if mimeType == "" {
		mimeType = "image/jpeg"
	}
	return &inboundImagePayload{
		Base64:   base64.StdEncoding.EncodeToString(data),
		MimeType: mimeType,
		Caption:  strings.TrimSpace(image.GetCaption()),
		Size:     len(data),
	}
}

func (rt *realTransport) persistStatus(ctx context.Context, status string, details map[string]any) error {
	sessionID := sessionIDFromContext(ctx)
	rt.mu.Lock()
	runtime := rt.legacy
	if sessionID != "" {
		runtime = rt.clients[sessionID]
	}
	if runtime != nil {
		runtime.lastStatus = normalizeSessionStatus(status)
		runtime.lastInfo = details
	}
	rt.mu.Unlock()
	return upsertStatus(ctx, rt.db, status, details)
}

func (rt *realTransport) runtimeBySession(sessionID string) *waSessionRuntime {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	if sessionID == "" {
		return rt.legacy
	}
	return rt.clients[sessionID]
}

func extractMessageText(message *waProto.Message) string {
	if message == nil {
		return ""
	}
	if text := strings.TrimSpace(message.GetConversation()); text != "" {
		return text
	}
	if text := strings.TrimSpace(message.GetExtendedTextMessage().GetText()); text != "" {
		return text
	}
	if text := strings.TrimSpace(message.GetImageMessage().GetCaption()); text != "" {
		return text
	}
	if text := strings.TrimSpace(message.GetVideoMessage().GetCaption()); text != "" {
		return text
	}
	return ""
}

func normalizePhone(phone string) string {
	phone = strings.TrimSpace(phone)
	phone = strings.TrimPrefix(phone, "+")
	phone = strings.TrimSuffix(phone, "@s.whatsapp.net")
	phone = strings.TrimSuffix(phone, "@lid")
	return phone
}

func recipientJID(recipient string) types.JID {
	recipient = strings.TrimSpace(recipient)
	lower := strings.ToLower(recipient)
	if strings.HasSuffix(lower, "@g.us") {
		return types.NewJID(strings.TrimSuffix(recipient, "@g.us"), "g.us")
	}
	return types.NewJID(normalizePhone(recipient), types.DefaultUserServer)
}

func isGroupJID(chatJID string) bool {
	return strings.HasSuffix(strings.ToLower(strings.TrimSpace(chatJID)), "@g.us")
}

func shouldIgnoreIncomingChat(chatJID string) bool {
	chatJID = strings.TrimSpace(strings.ToLower(chatJID))
	if chatJID == "" {
		return true
	}
	return strings.HasSuffix(chatJID, "@broadcast") ||
		chatJID == "status@broadcast"
}
