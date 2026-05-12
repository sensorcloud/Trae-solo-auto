package monitor

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type AlertStatus string
type AlertSeverity string
type AlertState string

const (
	AlertStatusActive    AlertStatus = "active"
	AlertStatusResolved  AlertStatus = "resolved"
	AlertStatusSilenced AlertStatus = "silenced"

	AlertSeverityCritical AlertSeverity = "critical"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityInfo     AlertSeverity = "info"

	AlertStateFiring   AlertState = "firing"
	AlertStatePending  AlertState = "pending"
	AlertStateResolved AlertState = "resolved"
)

type AlertRule struct {
	ID          uuid.UUID      `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Expr        string         `json:"expr"`
	For         time.Duration  `json:"for_duration"`
	Severity    AlertSeverity  `json:"severity"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
}

type Alert struct {
	ID          uuid.UUID     `json:"id"`
	RuleID      uuid.UUID     `json:"rule_id"`
	RuleName    string        `json:"rule_name"`
	Severity    AlertSeverity `json:"severity"`
	State       AlertState    `json:"state"`
	Status      AlertStatus   `json:"status"`
	Summary     string        `json:"summary"`
	Description string        `json:"description"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Value       float64       `json:"value"`
	Threshold   float64       `json:"threshold"`
	FiredAt     *time.Time    `json:"fired_at,omitempty"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
}

type AlertNotification struct {
	ID        uuid.UUID `json:"id"`
	AlertID   uuid.UUID `json:"alert_id"`
	Channel   string    `json:"channel"`
	Recipient string    `json:"recipient"`
	SentAt    time.Time `json:"sent_at"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
}

type MonitoringService struct {
	rules         map[uuid.UUID]*AlertRule
	alerts        map[uuid.UUID]*Alert
	notifications map[uuid.UUID]*AlertNotification
	evaluator     *RuleEvaluator
	mu            sync.RWMutex
}

func NewMonitoringService() *MonitoringService {
	return &MonitoringService{
		rules:         make(map[uuid.UUID]*AlertRule),
		alerts:        make(map[uuid.UUID]*Alert),
		notifications: make(map[uuid.UUID]*AlertNotification),
		evaluator:     NewRuleEvaluator(),
	}
}

func (ms *MonitoringService) CreateRule(ctx context.Context, rule *AlertRule) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	rule.ID = uuid.New()
	rule.CreatedAt = time.Now()
	rule.Enabled = true

	ms.rules[rule.ID] = rule
	klog.Infof("Created alert rule: %s (%s)", rule.Name, rule.ID)
	return nil
}

func (ms *MonitoringService) GetRule(ctx context.Context, ruleID uuid.UUID) (*AlertRule, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	rule, exists := ms.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("rule %s not found", ruleID)
	}
	return rule, nil
}

func (ms *MonitoringService) ListRules(ctx context.Context, filter *RuleFilter) ([]*AlertRule, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var result []*AlertRule
	for _, rule := range ms.rules {
		if filter != nil {
			if filter.Severity != "" && rule.Severity != filter.Severity {
				continue
			}
			if filter.Enabled != nil && rule.Enabled != *filter.Enabled {
				continue
			}
			if filter.Name != "" && rule.Name != filter.Name {
				continue
			}
		}
		result = append(result, rule)
	}
	return result, nil
}

func (ms *MonitoringService) UpdateRule(ctx context.Context, ruleID uuid.UUID, updates *RuleUpdate) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	rule, exists := ms.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	if updates.Name != nil {
		rule.Name = *updates.Name
	}
	if updates.Description != nil {
		rule.Description = *updates.Description
	}
	if updates.Expr != nil {
		rule.Expr = *updates.Expr
	}
	if updates.For != nil {
		rule.For = *updates.For
	}
	if updates.Severity != nil {
		rule.Severity = *updates.Severity
	}
	if updates.Labels != nil {
		rule.Labels = updates.Labels
	}
	if updates.Annotations != nil {
		rule.Annotations = updates.Annotations
	}
	if updates.Enabled != nil {
		rule.Enabled = *updates.Enabled
	}

	return nil
}

func (ms *MonitoringService) DeleteRule(ctx context.Context, ruleID uuid.UUID) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, exists := ms.rules[ruleID]; !exists {
		return fmt.Errorf("rule %s not found", ruleID)
	}

	delete(ms.rules, ruleID)
	klog.Infof("Deleted alert rule: %s", ruleID)
	return nil
}

func (ms *MonitoringService) GetAlert(ctx context.Context, alertID uuid.UUID) (*Alert, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	alert, exists := ms.alerts[alertID]
	if !exists {
		return nil, fmt.Errorf("alert %s not found", alertID)
	}
	return alert, nil
}

func (ms *MonitoringService) ListAlerts(ctx context.Context, filter *AlertFilter) ([]*Alert, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var result []*Alert
	for _, alert := range ms.alerts {
		if filter != nil {
			if filter.Status != "" && alert.Status != filter.Status {
				continue
			}
			if filter.State != "" && alert.State != filter.State {
				continue
			}
			if filter.Severity != "" && alert.Severity != filter.Severity {
				continue
			}
			if filter.RuleID != uuid.Nil && alert.RuleID != filter.RuleID {
				continue
			}
		}
		result = append(result, alert)
	}
	return result, nil
}

func (ms *MonitoringService) FireAlert(ctx context.Context, rule *AlertRule, value float64) (*Alert, error) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	now := time.Now()
	alert := &Alert{
		ID:          uuid.New(),
		RuleID:      rule.ID,
		RuleName:    rule.Name,
		Severity:    rule.Severity,
		State:       AlertStateFiring,
		Status:      AlertStatusActive,
		Summary:     rule.Annotations["summary"],
		Description: rule.Annotations["description"],
		Labels:      rule.Labels,
		Annotations: rule.Annotations,
		Value:       value,
		Threshold:   ms.extractThreshold(rule.Expr),
		FiredAt:     &now,
		CreatedAt:   now,
	}

	ms.alerts[alert.ID] = alert
	klog.Warningf("Alert fired: %s (%s) - value: %.2f", alert.RuleName, alert.ID, value)

	return alert, nil
}

func (ms *MonitoringService) ResolveAlert(ctx context.Context, alertID uuid.UUID) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	alert, exists := ms.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	now := time.Now()
	alert.Status = AlertStatusResolved
	alert.State = AlertStateResolved
	alert.ResolvedAt = &now

	klog.Infof("Alert resolved: %s (%s)", alert.RuleName, alertID)
	return nil
}

func (ms *MonitoringService) SilenceAlert(ctx context.Context, alertID uuid.UUID) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	alert, exists := ms.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	alert.Status = AlertStatusSilenced
	return nil
}

func (ms *MonitoringService) SendNotification(ctx context.Context, alertID uuid.UUID, channel, recipient string) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	notification := &AlertNotification{
		ID:        uuid.New(),
		AlertID:   alertID,
		Channel:   channel,
		Recipient: recipient,
		SentAt:    time.Now(),
	}

	notification.Success = true

	ms.notifications[notification.ID] = notification
	klog.Infof("Alert notification sent: %s via %s to %s", alertID, channel, recipient)

	return nil
}

func (ms *MonitoringService) GetActiveAlertCount(ctx context.Context) (int, map[AlertSeverity]int, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	count := 0
	bySeverity := map[AlertSeverity]int{
		AlertSeverityCritical: 0,
		AlertSeverityWarning:  0,
		AlertSeverityInfo:     0,
	}

	for _, alert := range ms.alerts {
		if alert.Status == AlertStatusActive {
			count++
			bySeverity[alert.Severity]++
		}
	}

	return count, bySeverity, nil
}

func (ms *MonitoringService) EvaluateRules(ctx context.Context, metrics map[string]float64) error {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	for _, rule := range ms.rules {
		if !rule.Enabled {
			continue
		}

		value := ms.evaluator.Evaluate(rule.Expr, metrics)
		if ms.evaluator.IsViolated(rule.Expr, value) {
			_, err := ms.FireAlert(ctx, rule, value)
			if err != nil {
				klog.Errorf("Failed to fire alert for rule %s: %v", rule.Name, err)
			}
		}
	}

	return nil
}

func (ms *MonitoringService) extractThreshold(expr string) float64 {
	return ms.evaluator.ExtractThreshold(expr)
}

type RuleEvaluator struct{}

func NewRuleEvaluator() *RuleEvaluator {
	return &RuleEvaluator{}
}

func (re *RuleEvaluator) Evaluate(expr string, metrics map[string]float64) float64 {
	if value, ok := metrics[expr]; ok {
		return value
	}
	return 0
}

func (re *RuleEvaluator) IsViolated(expr string, value float64) bool {
	return value > 0
}

func (re *RuleEvaluator) ExtractThreshold(expr string) float64 {
	return 0.8
}

type RuleFilter struct {
	Severity AlertSeverity
	Enabled  *bool
	Name     string
}

type RuleUpdate struct {
	Name        *string
	Description *string
	Expr        *string
	For         *time.Duration
	Severity    *AlertSeverity
	Labels      map[string]string
	Annotations map[string]string
	Enabled     *bool
}

type AlertFilter struct {
	Status   AlertStatus
	State    AlertState
	Severity AlertSeverity
	RuleID   uuid.UUID
}
