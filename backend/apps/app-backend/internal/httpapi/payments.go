package httpapi

import (
	"bytes"
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	errMidtransNotConfigured        = errors.New("midtrans is not configured")
	errCreditPurchaseNotFound       = errors.New("credit purchase not found")
	errCreditPurchaseNotConfirmable = errors.New("credit purchase is not confirmable")
)

type midtransSnapInput struct {
	PurchaseID    string
	OrderID       string
	PackageID     string
	PackageName   string
	CreditAmount  int
	Price         float64
	PaymentFee    int64
	PaymentMethod string
	CustomerName  string
}

type midtransSnapResult struct {
	Token       string
	RedirectURL string
	OrderID     string
	GrossAmount int64
}

type purchasePaymentQuote struct {
	PackageAmount int64
	PaymentFee    int64
	GrossAmount   int64
	FeeLabel      string
}

const (
	paymentGatewayVATRate       = 0.11
	virtualAccountGatewayFeeIDR = 4000
	qrisGatewayFeeRate          = 0.007
	cardGatewayFeeRate          = 0.029
	cardGatewayFlatFeeIDR       = 2000
)

func calculatePurchasePaymentQuote(price float64, method string) purchasePaymentQuote {
	packageAmount := int64(math.Round(price))
	if packageAmount < 0 {
		packageAmount = 0
	}

	var paymentFee int64
	feeLabel := "Biaya gateway"
	switch normalizePurchasePaymentMethod(method) {
	case "qris":
		grossAmount := ceilIDR(float64(packageAmount) / (1 - qrisGatewayFeeRate))
		paymentFee = grossAmount - packageAmount
		feeLabel = "Biaya gateway QRIS"
	case "card":
		flatFee := flatFeeWithVAT(cardGatewayFlatFeeIDR)
		percentWithVAT := cardGatewayFeeRate * (1 + paymentGatewayVATRate)
		grossAmount := ceilIDR((float64(packageAmount) + float64(flatFee)) / (1 - percentWithVAT))
		paymentFee = grossAmount - packageAmount
		feeLabel = "Biaya gateway kartu"
	default:
		paymentFee = flatFeeWithVAT(virtualAccountGatewayFeeIDR)
		feeLabel = "Biaya gateway Virtual Account"
	}
	if paymentFee < 0 {
		paymentFee = 0
	}
	return purchasePaymentQuote{
		PackageAmount: packageAmount,
		PaymentFee:    paymentFee,
		GrossAmount:   packageAmount + paymentFee,
		FeeLabel:      feeLabel,
	}
}

func flatFeeWithVAT(amount int64) int64 {
	return ceilIDR(float64(amount) * (1 + paymentGatewayVATRate))
}

func ceilIDR(amount float64) int64 {
	if amount <= 0 {
		return 0
	}
	return int64(math.Ceil(amount))
}

type midtransNotification struct {
	OrderID             string `json:"order_id"`
	StatusCode          string `json:"status_code"`
	GrossAmount         string `json:"gross_amount"`
	SignatureKey        string `json:"signature_key"`
	TransactionID       string `json:"transaction_id"`
	TransactionStatus   string `json:"transaction_status"`
	FraudStatus         string `json:"fraud_status"`
	PaymentType         string `json:"payment_type"`
	TransactionTime     string `json:"transaction_time"`
	SettlementTime      string `json:"settlement_time"`
	ExpiryTime          string `json:"expiry_time"`
	ExpireTime          string `json:"expire_time"`
	TransactionTimeISO  string `json:"transaction_time_iso"`
	SettlementTimeISO   string `json:"settlement_time_iso"`
	ExpiryTimeISO       string `json:"expiry_time_iso"`
	TransactionStatusID string `json:"transaction_status_id"`
}

func (s *Server) midtransConfigured() bool {
	return strings.TrimSpace(s.cfg.MidtransClientKey) != "" && strings.TrimSpace(s.cfg.MidtransServerKey) != ""
}

func (s *Server) midtransSnapEndpoint() string {
	if strings.EqualFold(s.cfg.MidtransEnv, "production") {
		return "https://app.midtrans.com/snap/v1/transactions"
	}
	return "https://app.sandbox.midtrans.com/snap/v1/transactions"
}

func (s *Server) midtransStatusEndpoint(orderID string) string {
	base := "https://api.sandbox.midtrans.com/v2"
	if strings.EqualFold(s.cfg.MidtransEnv, "production") {
		base = "https://api.midtrans.com/v2"
	}
	return base + "/" + url.PathEscape(orderID) + "/status"
}

func (s *Server) midtransCancelEndpoint(orderID string) string {
	base := "https://api.sandbox.midtrans.com/v2"
	if strings.EqualFold(s.cfg.MidtransEnv, "production") {
		base = "https://api.midtrans.com/v2"
	}
	return base + "/" + url.PathEscape(orderID) + "/cancel"
}

func (s *Server) createMidtransSnapTransaction(ctx context.Context, input midtransSnapInput) (*midtransSnapResult, error) {
	if !s.midtransConfigured() {
		return nil, errMidtransNotConfigured
	}

	grossAmount := int64(math.Round(input.Price))
	if grossAmount <= 0 {
		return nil, fmt.Errorf("purchase price must be positive")
	}
	packageAmount := grossAmount - input.PaymentFee
	if packageAmount <= 0 {
		packageAmount = grossAmount
	}

	itemID := strings.TrimSpace(input.PackageID)
	if len(itemID) > 50 {
		itemID = itemID[:50]
	}
	itemName := strings.TrimSpace(input.PackageName)
	if itemName == "" {
		itemName = "Oneflow.id package"
	}
	if len(itemName) > 50 {
		itemName = itemName[:50]
	}

	itemDetails := []map[string]any{
		{
			"id":       itemID,
			"price":    packageAmount,
			"quantity": 1,
			"name":     itemName,
		},
	}
	if input.PaymentFee > 0 {
		itemDetails = append(itemDetails, map[string]any{
			"id":       "gateway-fee",
			"price":    input.PaymentFee,
			"quantity": 1,
			"name":     "Biaya metode pembayaran",
		})
	}

	payload := map[string]any{
		"transaction_details": map[string]any{
			"order_id":     input.OrderID,
			"gross_amount": grossAmount,
		},
		"item_details": itemDetails,
		"callbacks": map[string]string{
			"finish":   s.cfg.AppPublicURL + "/dashboard/upgrade",
			"unfinish": s.cfg.AppPublicURL + "/dashboard/upgrade",
			"error":    s.cfg.AppPublicURL + "/dashboard/upgrade",
		},
	}

	customerName := strings.TrimSpace(input.CustomerName)
	if customerName != "" {
		payload["customer_details"] = map[string]string{"first_name": truncateMidtransValue(customerName, 255)}
	}
	if enabledPayments := midtransEnabledPayments(input.PaymentMethod); len(enabledPayments) > 0 {
		payload["enabled_payments"] = enabledPayments
	}
	if input.PaymentMethod == "card" {
		payload["credit_card"] = map[string]bool{"secure": true}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.midtransSnapEndpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(s.cfg.MidtransServerKey + ":"))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request midtrans snap: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var snapResp struct {
		Token         string   `json:"token"`
		RedirectURL   string   `json:"redirect_url"`
		ErrorMessages []string `json:"error_messages"`
		StatusMessage string   `json:"status_message"`
	}
	_ = json.Unmarshal(respBody, &snapResp)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(snapResp.StatusMessage)
		if message == "" && len(snapResp.ErrorMessages) > 0 {
			message = strings.Join(snapResp.ErrorMessages, "; ")
		}
		if message == "" {
			message = "Midtrans Snap request failed"
		}
		return nil, fmt.Errorf("%s", message)
	}
	if strings.TrimSpace(snapResp.Token) == "" || strings.TrimSpace(snapResp.RedirectURL) == "" {
		return nil, fmt.Errorf("Midtrans Snap response is missing token")
	}

	return &midtransSnapResult{
		Token:       snapResp.Token,
		RedirectURL: snapResp.RedirectURL,
		OrderID:     input.OrderID,
		GrossAmount: grossAmount,
	}, nil
}

func newMidtransOrderID(purchaseID string) string {
	compact := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(purchaseID), "-", ""))
	if compact == "" {
		compact = strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	if len(compact) > 24 {
		compact = compact[:24]
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	if len(suffix) > 10 {
		suffix = suffix[len(suffix)-10:]
	}
	return fmt.Sprintf("ONEFLOW-%s-%s", compact, suffix)
}

func (s *Server) saveMidtransPaymentOrder(ctx context.Context, purchaseID string, snapResult *midtransSnapResult, paymentMethod string) error {
	if snapResult == nil || strings.TrimSpace(snapResult.OrderID) == "" {
		return nil
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO credit_purchase_payment_orders (
		  credit_purchase_id, midtrans_order_id, payment_method, snap_token, snap_redirect_url, transaction_status
		)
		VALUES ($1, $2, $3, $4, $5, 'pending')
		ON CONFLICT (midtrans_order_id) DO UPDATE
		SET payment_method = EXCLUDED.payment_method,
		    snap_token = EXCLUDED.snap_token,
		    snap_redirect_url = EXCLUDED.snap_redirect_url,
		    transaction_status = EXCLUDED.transaction_status
	`, purchaseID, snapResult.OrderID, paymentMethod, snapResult.Token, snapResult.RedirectURL)
	return err
}

func (s *Server) fetchMidtransTransactionStatus(ctx context.Context, orderID string) (*midtransNotification, []byte, error) {
	if !s.midtransConfigured() {
		return nil, nil, errMidtransNotConfigured
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, nil, errCreditPurchaseNotFound
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.midtransStatusEndpoint(orderID), nil)
	if err != nil {
		return nil, nil, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(s.cfg.MidtransServerKey + ":"))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request midtrans status: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, err
	}

	var notification midtransNotification
	_ = json.Unmarshal(respBody, &notification)
	var statusResp struct {
		ErrorMessages []string `json:"error_messages"`
		StatusMessage string   `json:"status_message"`
	}
	_ = json.Unmarshal(respBody, &statusResp)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(statusResp.StatusMessage)
		if message == "" && len(statusResp.ErrorMessages) > 0 {
			message = strings.Join(statusResp.ErrorMessages, "; ")
		}
		if message == "" {
			message = "Midtrans status request failed"
		}
		return nil, nil, fmt.Errorf("%s", message)
	}

	if strings.TrimSpace(notification.OrderID) == "" {
		notification.OrderID = orderID
	}
	return &notification, respBody, nil
}

func (s *Server) cancelMidtransTransaction(ctx context.Context, orderID string) (*midtransNotification, []byte, error) {
	if !s.midtransConfigured() {
		return nil, nil, errMidtransNotConfigured
	}
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, nil, errCreditPurchaseNotFound
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.midtransCancelEndpoint(orderID), nil)
	if err != nil {
		return nil, nil, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(s.cfg.MidtransServerKey + ":"))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+auth)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("request midtrans cancel: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, err
	}

	var notification midtransNotification
	_ = json.Unmarshal(respBody, &notification)
	var statusResp struct {
		ErrorMessages []string `json:"error_messages"`
		StatusMessage string   `json:"status_message"`
	}
	_ = json.Unmarshal(respBody, &statusResp)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		message := strings.TrimSpace(statusResp.StatusMessage)
		if message == "" && len(statusResp.ErrorMessages) > 0 {
			message = strings.Join(statusResp.ErrorMessages, "; ")
		}
		if message == "" {
			message = "Midtrans cancel request failed"
		}
		return nil, nil, fmt.Errorf("%s", message)
	}

	if strings.TrimSpace(notification.OrderID) == "" {
		notification.OrderID = orderID
	}
	if strings.TrimSpace(notification.TransactionStatus) == "" {
		notification.TransactionStatus = "cancel"
	}
	return &notification, respBody, nil
}

func (s *Server) handleMidtransNotification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	if !s.midtransConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payment gateway is not configured"})
		return
	}

	rawBody, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid notification body"})
		return
	}

	var notification midtransNotification
	if err := json.Unmarshal(rawBody, &notification); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid notification json"})
		return
	}
	if strings.TrimSpace(notification.OrderID) == "" || strings.TrimSpace(notification.SignatureKey) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid notification payload"})
		return
	}
	if !s.verifyMidtransSignature(notification) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid notification signature"})
		return
	}

	tx, err := s.db.Begin(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	defer tx.Rollback(r.Context())

	if err := s.applyMidtransNotificationTx(r.Context(), tx, notification, rawBody); err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, errCreditPurchaseNotFound) {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) syncMidtransPurchaseStatus(ctx context.Context, purchaseID string, organizationID string) (string, error) {
	if !s.midtransConfigured() {
		return "", errMidtransNotConfigured
	}

	var orderID string
	if err := s.db.QueryRow(ctx, `
		SELECT COALESCE(midtrans_order_id, '')
		FROM credit_purchases
		WHERE id = $1 AND organization_id = $2
	`, purchaseID, organizationID).Scan(&orderID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errCreditPurchaseNotFound
		}
		return "", err
	}

	notification, rawBody, err := s.fetchMidtransTransactionStatus(ctx, orderID)
	if err != nil {
		return "", err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	if err := s.applyMidtransNotificationTx(ctx, tx, *notification, rawBody); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}

	var paymentStatus string
	if err := s.db.QueryRow(ctx, `
		SELECT payment_status
		FROM credit_purchases
		WHERE id = $1 AND organization_id = $2
	`, purchaseID, organizationID).Scan(&paymentStatus); err != nil {
		return "", err
	}
	return paymentStatus, nil
}

func (s *Server) verifyMidtransSignature(notification midtransNotification) bool {
	input := notification.OrderID + notification.StatusCode + notification.GrossAmount + s.cfg.MidtransServerKey
	sum := sha512.Sum512([]byte(input))
	expected := hex.EncodeToString(sum[:])
	actual := strings.ToLower(strings.TrimSpace(notification.SignatureKey))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

func (s *Server) applyMidtransNotificationTx(ctx context.Context, tx pgx.Tx, notification midtransNotification, rawBody []byte) error {
	var purchaseID, currentMidtransOrderID string
	if err := tx.QueryRow(ctx, `
		SELECT p.id, COALESCE(p.midtrans_order_id, '')
		FROM credit_purchases p
		LEFT JOIN credit_purchase_payment_orders po ON po.credit_purchase_id = p.id
		WHERE p.midtrans_order_id = $1 OR po.midtrans_order_id = $1
		ORDER BY CASE WHEN p.midtrans_order_id = $1 THEN 0 ELSE 1 END
		LIMIT 1
		FOR UPDATE OF p
	`, notification.OrderID).Scan(&purchaseID, &currentMidtransOrderID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errCreditPurchaseNotFound
		}
		return err
	}

	paidAt := midtransPaidAt(notification)
	expiredAt := parseMidtransTime(firstNonEmpty(notification.ExpiryTimeISO, notification.ExpiryTime, notification.ExpireTime))
	isCurrentOrder := notification.OrderID == currentMidtransOrderID
	if isCurrentOrder || midtransNotificationSuccessful(notification) {
		if _, err := tx.Exec(ctx, `
			UPDATE credit_purchases
			SET midtrans_transaction_id = COALESCE(NULLIF($2, ''), midtrans_transaction_id),
			    transaction_status = COALESCE(NULLIF($3, ''), transaction_status),
			    fraud_status = COALESCE(NULLIF($4, ''), fraud_status),
			    payment_type = COALESCE(NULLIF($5, ''), payment_type),
			    gross_amount = COALESCE(NULLIF($6, '')::numeric, gross_amount),
			    raw_notification = $7::jsonb,
			    paid_at = COALESCE($8, paid_at),
			    expired_at = COALESCE($9, expired_at)
			WHERE id = $1
		`, purchaseID, notification.TransactionID, notification.TransactionStatus, notification.FraudStatus, notification.PaymentType, notification.GrossAmount, string(rawBody), paidAt, expiredAt); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE credit_purchase_payment_orders
		SET transaction_status = COALESCE(NULLIF($2, ''), transaction_status)
		WHERE midtrans_order_id = $1
	`, notification.OrderID, notification.TransactionStatus); err != nil {
		return err
	}

	if midtransNotificationSuccessful(notification) {
		return s.confirmCreditPurchaseTx(ctx, tx, purchaseID, nil, paidAt, "billing.midtrans_settlement", true)
	}
	if !isCurrentOrder {
		return nil
	}

	paymentStatus := midtransPurchaseStatus(notification)
	if _, err := tx.Exec(ctx, `
		UPDATE credit_purchases
		SET payment_status = CASE
		  WHEN payment_status = 'confirmed' THEN payment_status
		  ELSE $2
		END
		WHERE id = $1
	`, purchaseID, paymentStatus); err != nil {
		return err
	}
	return nil
}

func (s *Server) confirmCreditPurchaseTx(ctx context.Context, tx pgx.Tx, id string, confirmedBy any, paidAt *time.Time, auditAction string, allowGatewaySettlement bool) error {
	var creditAmount int
	var organizationID, billingPeriod, paymentStatus string
	err := tx.QueryRow(ctx, `
		SELECT p.credit_amount, p.organization_id::text, COALESCE(p.billing_period, cp.billing_period, 'monthly'), p.payment_status
		FROM credit_purchases p
		LEFT JOIN credit_packages cp ON cp.id = p.credit_package_id AND cp.organization_id = p.organization_id
		WHERE p.id = $1
		FOR UPDATE OF p
	`, id).Scan(&creditAmount, &organizationID, &billingPeriod, &paymentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errCreditPurchaseNotFound
		}
		return err
	}
	if paymentStatus == "confirmed" {
		return nil
	}
	if allowGatewaySettlement {
		switch paymentStatus {
		case "cancelled", "canceled", "expired", "failed":
			return nil
		}
	}
	if !allowGatewaySettlement && paymentStatus != "requested" && paymentStatus != "pending" {
		return errCreditPurchaseNotConfirmable
	}

	var activeFrom, activeUntil time.Time
	subscriptionMonths := 1
	if billingPeriod == "annual" {
		subscriptionMonths = 12
	}
	if billingPeriod == "one_time" {
		err = tx.QueryRow(ctx, `
			SELECT
			  COALESCE($3::timestamptz, NOW()),
			  COALESCE(
			    MAX(COALESCE(p.active_until, p.confirmed_at + CASE WHEN COALESCE(p.billing_period, cp.billing_period, 'monthly') = 'annual' THEN INTERVAL '12 months' ELSE INTERVAL '1 month' END)),
			    COALESCE($3::timestamptz, NOW())
			  )
			FROM credit_purchases p
			JOIN credit_packages cp ON cp.id = p.credit_package_id AND cp.organization_id = p.organization_id
			WHERE p.organization_id = $1
			  AND p.id <> $2
			  AND p.payment_status = 'confirmed'
			  AND COALESCE(p.billing_period, cp.billing_period, 'monthly') IN ('monthly', 'annual')
			  AND COALESCE(p.active_until, p.confirmed_at + CASE WHEN COALESCE(p.billing_period, cp.billing_period, 'monthly') = 'annual' THEN INTERVAL '12 months' ELSE INTERVAL '1 month' END) > NOW()
		`, organizationID, id, paidAt).Scan(&activeFrom, &activeUntil)
	} else {
		err = tx.QueryRow(ctx, `
			SELECT
			  COALESCE($3::timestamptz, NOW()),
			  GREATEST(
			    COALESCE($3::timestamptz, NOW()),
			    COALESCE(MAX(COALESCE(p.active_until, p.confirmed_at + CASE WHEN COALESCE(p.billing_period, cp.billing_period, 'monthly') = 'annual' THEN INTERVAL '12 months' ELSE INTERVAL '1 month' END)), COALESCE($3::timestamptz, NOW()))
			  ) + ($4 * INTERVAL '1 month')
			FROM credit_purchases p
			JOIN credit_packages cp ON cp.id = p.credit_package_id AND cp.organization_id = p.organization_id
			WHERE p.organization_id = $1
			  AND p.id <> $2
			  AND p.payment_status = 'confirmed'
			  AND COALESCE(p.billing_period, cp.billing_period, 'monthly') IN ('monthly', 'annual')
			  AND COALESCE(p.active_until, p.confirmed_at + CASE WHEN COALESCE(p.billing_period, cp.billing_period, 'monthly') = 'annual' THEN INTERVAL '12 months' ELSE INTERVAL '1 month' END) > NOW()
		`, organizationID, id, paidAt, subscriptionMonths).Scan(&activeFrom, &activeUntil)
	}
	if err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `
		UPDATE credit_purchases
		SET payment_status = 'confirmed',
		    confirmed_by = COALESCE($2::uuid, confirmed_by),
		    confirmed_at = COALESCE(confirmed_at, NOW()),
		    paid_at = COALESCE($3, paid_at, NOW()),
		    active_from = COALESCE(active_from, $5),
		    active_until = COALESCE(active_until, $6)
		WHERE id = $1 AND organization_id = $4
	`, id, confirmedBy, paidAt, organizationID, activeFrom, activeUntil); err != nil {
		return err
	}

	if billingPeriod == "one_time" {
		_, err = tx.Exec(ctx, `
			UPDATE credit_wallet
			SET additional_credits_remaining = additional_credits_remaining + $1,
			    updated_at = NOW()
			WHERE is_active = TRUE AND organization_id = $2
		`, creditAmount, organizationID)
	} else {
		nextResetAt := activeUntil
		if billingPeriod == "annual" {
			nextResetAt = activeFrom.AddDate(0, 1, 0)
		}
		_, err = tx.Exec(ctx, `
			UPDATE credit_wallet
			SET monthly_credit_limit = $1,
			    monthly_credits_used = 0,
			    monthly_credits_remaining = $1,
			    last_reset_at = $3,
			    next_reset_at = $4,
			    updated_at = NOW()
			WHERE is_active = TRUE AND organization_id = $2
		`, creditAmount, organizationID, activeFrom, nextResetAt)
	}
	if err != nil {
		return err
	}

	if auditAction == "" {
		auditAction = "billing.purchase_confirm"
	}
	return s.insertAuditLogTx(ctx, tx, auditAction, "credit_purchase", id, map[string]any{
		"credit_amount":  creditAmount,
		"billing_period": billingPeriod,
		"active_until":   activeUntil,
	})
}

func midtransNotificationSuccessful(notification midtransNotification) bool {
	status := strings.ToLower(strings.TrimSpace(notification.TransactionStatus))
	fraud := strings.ToLower(strings.TrimSpace(notification.FraudStatus))
	if status == "settlement" {
		return true
	}
	if status == "capture" {
		return fraud == "" || fraud == "accept"
	}
	return false
}

func midtransPurchaseStatus(notification midtransNotification) string {
	switch strings.ToLower(strings.TrimSpace(notification.TransactionStatus)) {
	case "pending":
		return "pending"
	case "expire":
		return "expired"
	case "cancel":
		return "cancelled"
	case "deny", "failure":
		return "failed"
	case "refund", "partial_refund":
		return "refunded"
	case "capture":
		if strings.EqualFold(notification.FraudStatus, "challenge") {
			return "pending"
		}
		return "failed"
	default:
		return "pending"
	}
}

func midtransPaidAt(notification midtransNotification) *time.Time {
	return parseMidtransTime(firstNonEmpty(
		notification.SettlementTimeISO,
		notification.SettlementTime,
		notification.TransactionTimeISO,
		notification.TransactionTime,
	))
}

func parseMidtransTime(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05 -0700 MST",
	}
	for _, layout := range layouts {
		if layout == "2006-01-02 15:04:05" {
			if parsed, err := time.ParseInLocation(layout, value, time.FixedZone("WIB", 7*60*60)); err == nil {
				return &parsed
			}
			continue
		}
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed
		}
	}
	return nil
}

func midtransEnabledPayments(method string) []string {
	switch normalizePurchasePaymentMethod(method) {
	case "qris":
		return []string{"qris"}
	case "card":
		return []string{"credit_card"}
	case "manual_transfer", "virtual_account", "bank_transfer":
		return []string{"bank_transfer", "bca_va", "bni_va", "bri_va", "permata_va", "echannel"}
	default:
		return nil
	}
}

func normalizePurchasePaymentMethod(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "qris":
		return "qris"
	case "card", "credit_card":
		return "card"
	case "manual_transfer", "virtual_account", "bank_transfer":
		return "manual_transfer"
	default:
		return "manual_transfer"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncateMidtransValue(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
