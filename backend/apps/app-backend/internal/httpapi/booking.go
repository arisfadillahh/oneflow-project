package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type bookingServiceRequest struct {
	Name            *string        `json:"name"`
	Description     *string        `json:"description"`
	DurationMinutes *int           `json:"durationMinutes"`
	BufferMinutes   *int           `json:"bufferMinutes"`
	Price           *float64       `json:"price"`
	Timezone        *string        `json:"timezone"`
	Availability    map[string]any `json:"availability"`
	Status          *string        `json:"status"`
}

type bookingAppointmentRequest struct {
	ServiceID       *string `json:"serviceId"`
	ContactID       *string `json:"contactId"`
	ConversationID  *string `json:"conversationId"`
	CustomerName    *string `json:"customerName"`
	CustomerPhone   *string `json:"customerPhone"`
	ScheduledStart  *string `json:"scheduledStart"`
	Status          *string `json:"status"`
	Source          *string `json:"source"`
	Notes           *string `json:"notes"`
	LocationType    *string `json:"locationType"`
	LocationAddress *string `json:"locationAddress"`
	LocationNotes   *string `json:"locationNotes"`
}

type bookingServiceSchedule struct {
	DurationMinutes int
	BufferMinutes   int
	Timezone        string
	Status          string
	Availability    map[string]any
}

func canManageBookingServices(role string) bool {
	return role == "owner" || role == "super_admin" || role == "admin"
}

func canManageBookingAppointments(role string) bool {
	return role == "owner" || isOpsRole(role)
}

func (s *Server) handleBookingServices(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "booking") {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.listBookingServices(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "canManage": canManageBookingServices(s.role(r.Context()))})
	case http.MethodPost:
		if !canManageBookingServices(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage booking services"})
			return
		}
		var req bookingServiceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createBookingService(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleBookingServiceRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "booking") {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/booking/services/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		if !canManageBookingServices(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only admin can manage booking services"})
			return
		}
		var req bookingServiceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.updateBookingService(r.Context(), id, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleBookingAppointments(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "booking") {
		return
	}
	switch r.Method {
	case http.MethodGet:
		items, err := s.listBookingAppointments(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")))
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	case http.MethodPost:
		if !canManageBookingAppointments(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only operational roles can manage booking"})
			return
		}
		var req bookingAppointmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.createBookingAppointment(r.Context(), req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleBookingAppointmentRoutes(w http.ResponseWriter, r *http.Request) {
	if !s.requireBusinessModule(w, r, "booking") {
		return
	}
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/booking/appointments/"), "/")
	if id == "" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		if !canManageBookingAppointments(s.role(r.Context())) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "only operational roles can manage booking"})
			return
		}
		var req bookingAppointmentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
			return
		}
		item, err := s.updateBookingAppointment(r.Context(), id, req)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"item": item})
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) listBookingServices(ctx context.Context, status string) ([]map[string]any, error) {
	if status == "" {
		status = "all"
	}
	rows, err := s.db.Query(ctx, `
		SELECT id::text, name, COALESCE(description, ''), duration_minutes, buffer_minutes,
		       price::float8, timezone, availability, status, created_at, updated_at
		FROM booking_services
		WHERE organization_id = $1
		  AND ($2 = 'all' OR status = $2)
		ORDER BY updated_at DESC
		LIMIT 150
	`, s.organizationID(ctx), status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		item, err := scanBookingService(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) createBookingService(ctx context.Context, req bookingServiceRequest) (map[string]any, error) {
	name := strings.TrimSpace(valueOrEmpty(req.Name))
	if name == "" {
		return nil, errors.New("name is required")
	}
	duration := 30
	if req.DurationMinutes != nil {
		duration = *req.DurationMinutes
	}
	buffer := 0
	if req.BufferMinutes != nil {
		buffer = *req.BufferMinutes
	}
	price := 0.0
	if req.Price != nil {
		price = *req.Price
	}
	if duration <= 0 || duration > 1440 {
		return nil, errors.New("duration must be between 1 and 1440 minutes")
	}
	if buffer < 0 || buffer > 1440 {
		return nil, errors.New("buffer must be between 0 and 1440 minutes")
	}
	if price < 0 {
		return nil, errors.New("price cannot be negative")
	}
	timezone := strings.TrimSpace(valueOrDefault(req.Timezone, "Asia/Jakarta"))
	status := normalizeBookingServiceStatus(valueOrEmpty(req.Status))
	availability := defaultBookingAvailability()
	if req.Availability != nil {
		availability = req.Availability
	}

	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO booking_services (
		  organization_id, name, description, duration_minutes, buffer_minutes, price,
		  timezone, availability, status, created_by, updated_by
		)
		VALUES ($1, $2, NULLIF($3, ''), $4, $5, $6, $7, $8::jsonb, $9, NULLIF($10, '')::uuid, NULLIF($10, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), name, strings.TrimSpace(valueOrEmpty(req.Description)), duration, buffer, price, timezone, marshalJSON(availability), status, s.agentID(ctx)).Scan(&id)
	if err != nil {
		return nil, err
	}
	_ = s.insertAuditLog(ctx, "booking.service_create", "booking_service", id, map[string]any{"name": name, "durationMinutes": duration})
	return s.bookingServiceByID(ctx, id)
}

func (s *Server) updateBookingService(ctx context.Context, id string, req bookingServiceRequest) (map[string]any, error) {
	current, err := s.bookingServiceByID(ctx, id)
	if err != nil {
		return nil, errors.New("booking service not found")
	}
	name := strings.TrimSpace(stringFromMap(current, "name"))
	description := strings.TrimSpace(stringFromMap(current, "description"))
	duration := intFromMap(current, "durationMinutes")
	buffer := intFromMap(current, "bufferMinutes")
	price := floatFromMap(current, "price")
	timezone := strings.TrimSpace(stringFromMap(current, "timezone"))
	status := strings.TrimSpace(stringFromMap(current, "status"))
	availability := mapFromAny(current["availability"])

	if req.Name != nil {
		name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		description = strings.TrimSpace(*req.Description)
	}
	if req.DurationMinutes != nil {
		duration = *req.DurationMinutes
	}
	if req.BufferMinutes != nil {
		buffer = *req.BufferMinutes
	}
	if req.Price != nil {
		price = *req.Price
	}
	if req.Timezone != nil {
		timezone = strings.TrimSpace(*req.Timezone)
	}
	if req.Status != nil {
		status = normalizeBookingServiceStatus(*req.Status)
	}
	if req.Availability != nil {
		availability = req.Availability
	}
	if name == "" {
		return nil, errors.New("name is required")
	}
	if duration <= 0 || duration > 1440 {
		return nil, errors.New("duration must be between 1 and 1440 minutes")
	}
	if buffer < 0 || buffer > 1440 {
		return nil, errors.New("buffer must be between 0 and 1440 minutes")
	}
	if price < 0 {
		return nil, errors.New("price cannot be negative")
	}
	if timezone == "" {
		timezone = "Asia/Jakarta"
	}

	_, err = s.db.Exec(ctx, `
		UPDATE booking_services
		SET name = $2,
		    description = NULLIF($3, ''),
		    duration_minutes = $4,
		    buffer_minutes = $5,
		    price = $6,
		    timezone = $7,
		    availability = $8::jsonb,
		    status = $9,
		    updated_by = NULLIF($10, '')::uuid,
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $11
	`, id, name, description, duration, buffer, price, timezone, marshalJSON(availability), status, s.agentID(ctx), s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	_ = s.insertAuditLog(ctx, "booking.service_update", "booking_service", id, map[string]any{"name": name, "status": status})
	return s.bookingServiceByID(ctx, id)
}

func (s *Server) bookingServiceByID(ctx context.Context, id string) (map[string]any, error) {
	row := s.db.QueryRow(ctx, `
		SELECT id::text, name, COALESCE(description, ''), duration_minutes, buffer_minutes,
		       price::float8, timezone, availability, status, created_at, updated_at
		FROM booking_services
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, id, s.organizationID(ctx))
	return scanBookingService(row)
}

func scanBookingService(row interface {
	Scan(dest ...any) error
}) (map[string]any, error) {
	var id, name, description, timezone, status string
	var duration, buffer int
	var price float64
	var availabilityRaw []byte
	var createdAt, updatedAt any
	if err := row.Scan(&id, &name, &description, &duration, &buffer, &price, &timezone, &availabilityRaw, &status, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":              id,
		"name":            name,
		"description":     description,
		"durationMinutes": duration,
		"bufferMinutes":   buffer,
		"price":           price,
		"timezone":        timezone,
		"availability":    jsonMapFromBytes(availabilityRaw),
		"status":          status,
		"createdAt":       createdAt,
		"updatedAt":       updatedAt,
	}, nil
}

func (s *Server) listBookingAppointments(ctx context.Context, status string) ([]map[string]any, error) {
	if status == "" {
		status = "all"
	}
	rows, err := s.db.Query(ctx, `
		SELECT a.id::text, a.service_id::text, s.name, COALESCE(a.contact_id::text, ''),
		       COALESCE(a.conversation_id::text, ''), COALESCE(a.customer_name, ''),
		       COALESCE(a.customer_phone, ''), a.scheduled_start, a.scheduled_end,
		       a.status, a.source, COALESCE(a.notes, ''), a.location_type,
		       COALESCE(a.location_address, ''), COALESCE(a.location_notes, ''),
		       a.created_at, a.updated_at
		FROM booking_appointments a
		JOIN booking_services s ON s.id = a.service_id AND s.organization_id = a.organization_id
		WHERE a.organization_id = $1
		  AND ($2 = 'all' OR a.status = $2)
		ORDER BY a.scheduled_start DESC
		LIMIT 150
	`, s.organizationID(ctx), status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		item, err := scanBookingAppointment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Server) createBookingAppointment(ctx context.Context, req bookingAppointmentRequest) (map[string]any, error) {
	serviceID := strings.TrimSpace(valueOrEmpty(req.ServiceID))
	if serviceID == "" {
		return nil, errors.New("serviceId is required")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	schedule, err := s.lockBookingService(ctx, tx, serviceID)
	if err != nil {
		return nil, errors.New("booking service not found")
	}
	if schedule.Status != "active" {
		return nil, errors.New("booking service is inactive")
	}
	scheduledStart, err := parseBookingTime(valueOrEmpty(req.ScheduledStart), schedule.Timezone)
	if err != nil {
		return nil, err
	}
	scheduledEnd := scheduledStart.Add(time.Duration(schedule.DurationMinutes) * time.Minute)
	status := normalizeBookingAppointmentStatus(valueOrDefault(req.Status, "scheduled"))
	if bookingAppointmentNeedsSlot(status) {
		if err := validateBookingSchedule(schedule, scheduledStart, scheduledEnd, time.Now()); err != nil {
			return nil, err
		}
		conflict, err := bookingConflictExists(ctx, tx, s.organizationID(ctx), serviceID, scheduledStart, scheduledEnd, "", schedule.BufferMinutes)
		if err != nil {
			return nil, err
		}
		if conflict {
			return nil, errors.New("booking schedule already has an appointment")
		}
	}
	contactID := strings.TrimSpace(valueOrEmpty(req.ContactID))
	if contactID != "" && !s.contactExists(ctx, contactID) {
		return nil, errors.New("contact not found")
	}
	conversationID := strings.TrimSpace(valueOrEmpty(req.ConversationID))
	if conversationID != "" && !s.conversationBelongsToOrganization(ctx, conversationID) {
		return nil, errors.New("conversation not found")
	}
	source := strings.TrimSpace(valueOrDefault(req.Source, "dashboard"))
	locationType := normalizeBookingLocationType(valueOrEmpty(req.LocationType))
	locationAddress := strings.TrimSpace(valueOrEmpty(req.LocationAddress))
	locationNotes := strings.TrimSpace(valueOrEmpty(req.LocationNotes))
	if err := validateBookingLocation(locationType, locationAddress); err != nil {
		return nil, err
	}

	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO booking_appointments (
		  organization_id, service_id, contact_id, conversation_id, customer_name, customer_phone,
		  scheduled_start, scheduled_end, status, source, notes, location_type, location_address, location_notes, created_by
		)
		VALUES ($1, NULLIF($2, '')::uuid, NULLIF($3, '')::uuid, NULLIF($4, '')::uuid,
		        NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9, $10, NULLIF($11, ''), $12,
		        NULLIF($13, ''), NULLIF($14, ''), NULLIF($15, '')::uuid)
		RETURNING id::text
	`, s.organizationID(ctx), serviceID, contactID, conversationID, strings.TrimSpace(valueOrEmpty(req.CustomerName)), strings.TrimSpace(valueOrEmpty(req.CustomerPhone)), scheduledStart, scheduledEnd, status, source, strings.TrimSpace(valueOrEmpty(req.Notes)), locationType, locationAddress, locationNotes, s.agentID(ctx)).Scan(&id)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = s.insertAuditLog(ctx, "booking.appointment_create", "booking_appointment", id, map[string]any{"serviceId": serviceID, "status": status})
	item, err := s.bookingAppointmentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.notifyBusinessPluginGroup(ctx, groupNotificationTriggerBookingCreated, item)
	return item, nil
}

func (s *Server) updateBookingAppointment(ctx context.Context, id string, req bookingAppointmentRequest) (map[string]any, error) {
	current, err := s.bookingAppointmentByID(ctx, id)
	if err != nil {
		return nil, errors.New("booking appointment not found")
	}
	serviceID := stringFromMap(current, "serviceId")
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	schedule, err := s.lockBookingService(ctx, tx, serviceID)
	if err != nil {
		return nil, errors.New("booking service not found")
	}
	// Another update may have committed while this request waited for the service lock.
	current, err = s.bookingAppointmentByID(ctx, id)
	if err != nil {
		return nil, errors.New("booking appointment not found")
	}
	status := stringFromMap(current, "status")
	notes := stringFromMap(current, "notes")
	locationType := stringFromMap(current, "locationType")
	locationAddress := stringFromMap(current, "locationAddress")
	locationNotes := stringFromMap(current, "locationNotes")
	scheduledStart, err := timeFromAny(current["scheduledStart"])
	if err != nil {
		return nil, err
	}
	scheduledEnd, err := timeFromAny(current["scheduledEnd"])
	if err != nil {
		return nil, err
	}
	if req.Status != nil {
		status = normalizeBookingAppointmentStatus(*req.Status)
	}
	if req.Notes != nil {
		notes = strings.TrimSpace(*req.Notes)
	}
	if req.LocationType != nil {
		locationType = normalizeBookingLocationType(*req.LocationType)
	}
	if req.LocationAddress != nil {
		locationAddress = strings.TrimSpace(*req.LocationAddress)
	}
	if req.LocationNotes != nil {
		locationNotes = strings.TrimSpace(*req.LocationNotes)
	}
	if err := validateBookingLocation(locationType, locationAddress); err != nil {
		return nil, err
	}
	if req.ScheduledStart != nil {
		scheduledStart, err = parseBookingTime(*req.ScheduledStart, schedule.Timezone)
		if err != nil {
			return nil, err
		}
		scheduledEnd = scheduledStart.Add(time.Duration(schedule.DurationMinutes) * time.Minute)
	}
	if bookingAppointmentNeedsSlot(status) {
		if req.ScheduledStart != nil || !bookingAppointmentNeedsSlot(stringFromMap(current, "status")) {
			if err := validateBookingSchedule(schedule, scheduledStart, scheduledEnd, time.Now()); err != nil {
				return nil, err
			}
		}
		conflict, err := bookingConflictExists(ctx, tx, s.organizationID(ctx), serviceID, scheduledStart, scheduledEnd, id, schedule.BufferMinutes)
		if err != nil {
			return nil, err
		}
		if conflict {
			return nil, errors.New("booking schedule already has an appointment")
		}
	}

	_, err = tx.Exec(ctx, `
		UPDATE booking_appointments
		SET scheduled_start = $2,
		    scheduled_end = $3,
		    status = $4,
		    notes = NULLIF($5, ''),
		    location_type = $6,
		    location_address = NULLIF($7, ''),
		    location_notes = NULLIF($8, ''),
		    updated_at = NOW()
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $9
	`, id, scheduledStart, scheduledEnd, status, notes, locationType, locationAddress, locationNotes, s.organizationID(ctx))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = s.insertAuditLog(ctx, "booking.appointment_update", "booking_appointment", id, map[string]any{"status": status})
	item, err := s.bookingAppointmentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s.notifyBusinessPluginGroup(ctx, groupNotificationTriggerBookingUpdated, item)
	return item, nil
}

func (s *Server) bookingAppointmentByID(ctx context.Context, id string) (map[string]any, error) {
	row := s.db.QueryRow(ctx, `
		SELECT a.id::text, a.service_id::text, s.name, COALESCE(a.contact_id::text, ''),
		       COALESCE(a.conversation_id::text, ''), COALESCE(a.customer_name, ''),
		       COALESCE(a.customer_phone, ''), a.scheduled_start, a.scheduled_end,
		       a.status, a.source, COALESCE(a.notes, ''), a.location_type,
		       COALESCE(a.location_address, ''), COALESCE(a.location_notes, ''),
		       a.created_at, a.updated_at
		FROM booking_appointments a
		JOIN booking_services s ON s.id = a.service_id AND s.organization_id = a.organization_id
		WHERE a.id = NULLIF($1, '')::uuid AND a.organization_id = $2
	`, id, s.organizationID(ctx))
	return scanBookingAppointment(row)
}

func scanBookingAppointment(row interface {
	Scan(dest ...any) error
}) (map[string]any, error) {
	var id, serviceID, serviceName, contactID, conversationID, customerName, customerPhone, status, source, notes string
	var locationType, locationAddress, locationNotes string
	var scheduledStart, scheduledEnd time.Time
	var createdAt, updatedAt any
	if err := row.Scan(&id, &serviceID, &serviceName, &contactID, &conversationID, &customerName, &customerPhone, &scheduledStart, &scheduledEnd, &status, &source, &notes, &locationType, &locationAddress, &locationNotes, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":              id,
		"serviceId":       serviceID,
		"serviceName":     serviceName,
		"contactId":       contactID,
		"conversationId":  conversationID,
		"customerName":    customerName,
		"customerPhone":   customerPhone,
		"scheduledStart":  scheduledStart,
		"scheduledEnd":    scheduledEnd,
		"status":          status,
		"source":          source,
		"notes":           notes,
		"locationType":    locationType,
		"locationAddress": locationAddress,
		"locationNotes":   locationNotes,
		"createdAt":       createdAt,
		"updatedAt":       updatedAt,
	}, nil
}

func (s *Server) bookingServiceScheduleByID(ctx context.Context, serviceID string) (bookingServiceSchedule, error) {
	var item bookingServiceSchedule
	err := s.db.QueryRow(ctx, `
		SELECT duration_minutes, buffer_minutes, timezone, status
		FROM booking_services
		WHERE id = NULLIF($1, '')::uuid AND organization_id = $2
	`, serviceID, s.organizationID(ctx)).Scan(&item.DurationMinutes, &item.BufferMinutes, &item.Timezone, &item.Status)
	return item, err
}

func bookingConflictExists(ctx context.Context, tx pgx.Tx, organizationID, serviceID string, scheduledStart, scheduledEnd time.Time, excludeID string, bufferMinutes int) (bool, error) {
	var exists bool
	err := tx.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1
		  FROM booking_appointments
		  WHERE organization_id = $1
		    AND service_id = NULLIF($2, '')::uuid
		    AND status IN ('scheduled', 'confirmed')
		    AND ($5 = '' OR id <> NULLIF($5, '')::uuid)
		    AND scheduled_start < $4
		    AND scheduled_end > $3
		)
	`, organizationID, serviceID, scheduledStart.Add(-time.Duration(bufferMinutes)*time.Minute), scheduledEnd.Add(time.Duration(bufferMinutes)*time.Minute), strings.TrimSpace(excludeID)).Scan(&exists)
	return exists, err
}

// Lock the service row before reading conflicts so concurrent create/reschedule requests serialize.
func (s *Server) lockBookingService(ctx context.Context, tx pgx.Tx, serviceID string) (bookingServiceSchedule, error) {
	var item bookingServiceSchedule
	var availability []byte
	err := tx.QueryRow(ctx, `SELECT duration_minutes, buffer_minutes, timezone, status, availability
		FROM booking_services WHERE id = NULLIF($1, '')::uuid AND organization_id = $2 FOR UPDATE`,
		serviceID, s.organizationID(ctx)).Scan(&item.DurationMinutes, &item.BufferMinutes, &item.Timezone, &item.Status, &availability)
	item.Availability = jsonMapFromBytes(availability)
	return item, err
}

func validateBookingSchedule(schedule bookingServiceSchedule, start, end, now time.Time) error {
	if schedule.Status != "active" {
		return errors.New("booking service is inactive")
	}
	if !start.After(now) {
		return errors.New("booking must be scheduled in the future")
	}
	location, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		return errors.New("booking service timezone is invalid")
	}
	localStart, localEnd := start.In(location), end.In(location)
	availability := schedule.Availability
	if availability == nil {
		availability = defaultBookingAvailability()
	}
	rawDays, _ := json.Marshal(availability["days"])
	var days []int
	if json.Unmarshal(rawDays, &days) != nil {
		return errors.New("booking service days are invalid")
	}
	allowed := false
	for _, day := range days {
		if day == int(localStart.Weekday()) {
			allowed = true
		}
	}
	if !allowed {
		return errors.New("booking is outside service days")
	}
	opening, err := time.ParseInLocation("2006-01-02 15:04", localStart.Format("2006-01-02")+" "+stringFromAny(availability["start"]), location)
	if err != nil {
		return errors.New("booking service opening time is invalid")
	}
	closing, err := time.ParseInLocation("2006-01-02 15:04", localStart.Format("2006-01-02")+" "+stringFromAny(availability["end"]), location)
	if err != nil || !closing.After(opening) {
		return errors.New("booking service closing time is invalid")
	}
	if localStart.Before(opening) || localEnd.After(closing) || !end.After(start) {
		return errors.New("booking is outside service hours")
	}
	return nil
}

func normalizeBookingServiceStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "inactive", "archived":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "active"
	}
}

func normalizeBookingAppointmentStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "draft", "confirmed", "cancelled", "completed":
		return strings.ToLower(strings.TrimSpace(value))
	case "canceled":
		return "cancelled"
	default:
		return "scheduled"
	}
}

func bookingAppointmentNeedsSlot(status string) bool {
	return status == "scheduled" || status == "confirmed"
}

func normalizeBookingLocationType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "online", "customer_address":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "business_location"
	}
}

func validateBookingLocation(locationType, address string) error {
	if locationType == "customer_address" && strings.TrimSpace(address) == "" {
		return errors.New("location address is required for customer address booking")
	}
	return nil
}

func parseBookingTime(value, timezone string) (time.Time, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return time.Time{}, errors.New("scheduledStart is required")
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed.UTC(), nil
	}
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		location = time.UTC
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02T15:04:05", "2006-01-02 15:04", "2006-01-02 15:04:05"} {
		if parsed, err := time.ParseInLocation(layout, raw, location); err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, errors.New("scheduledStart has invalid format")
}

func timeFromAny(value any) (time.Time, error) {
	switch parsed := value.(type) {
	case time.Time:
		return parsed, nil
	case *time.Time:
		if parsed == nil {
			return time.Time{}, errors.New("time value is empty")
		}
		return *parsed, nil
	case string:
		return parseBookingTime(parsed, "UTC")
	default:
		return time.Time{}, errors.New("time value has invalid format")
	}
}

func defaultBookingAvailability() map[string]any {
	return map[string]any{
		"days":  []int{1, 2, 3, 4, 5},
		"start": "09:00",
		"end":   "17:00",
	}
}

func jsonMapFromBytes(raw []byte) map[string]any {
	var parsed map[string]any
	if len(raw) > 0 && json.Unmarshal(raw, &parsed) == nil && parsed != nil {
		return parsed
	}
	return defaultBookingAvailability()
}

func mapFromAny(value any) map[string]any {
	if parsed, ok := value.(map[string]any); ok && parsed != nil {
		return parsed
	}
	return defaultBookingAvailability()
}
