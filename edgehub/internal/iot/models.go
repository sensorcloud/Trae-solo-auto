package iot

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IoTBaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (b *IoTBaseModel) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

type DeviceStatus string

const (
	DeviceStatusOnline    DeviceStatus = "online"
	DeviceStatusOffline   DeviceStatus = "offline"
	DeviceStatusPending   DeviceStatus = "pending"
	DeviceStatusInactive  DeviceStatus = "inactive"
	DeviceStatusError     DeviceStatus = "error"
	DeviceStatusMaintain  DeviceStatus = "maintain"
)

type ProtocolType string

const (
	ProtocolMQTT   ProtocolType = "mqtt"
	ProtocolModbus ProtocolType = "modbus"
	ProtocolOPCUA  ProtocolType = "opcua"
	ProtocolHTTP   ProtocolType = "http"
	ProtocolCoAP   ProtocolType = "coap"
)

type DeviceType string

const (
	DeviceTypeSensor      DeviceType = "sensor"
	DeviceTypeActuator    DeviceType = "actuator"
	DeviceTypeGateway     DeviceType = "gateway"
	DeviceTypeController  DeviceType = "controller"
	DeviceTypeEnergyMeter DeviceType = "energy_meter"
	DeviceTypeComputeNode DeviceType = "compute_node"
	DeviceTypeSmartMeter  DeviceType = "smart_meter"
	DeviceTypeInverter    DeviceType = "inverter"
	DeviceTypeUPS         DeviceType = "ups"
	DeviceTypePDU         DeviceType = "pdu"
)

type Device struct {
	IoTBaseModel
	TenantID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"tenant_id"`
	Name            string         `gorm:"size:100;not null;index" json:"name"`
	DisplayName     string         `gorm:"size:200" json:"display_name"`
	Description     string         `gorm:"type:text" json:"description"`
	DeviceType      DeviceType     `gorm:"size:30;not null;index" json:"device_type"`
	Status          DeviceStatus   `gorm:"size:20;default:'pending';index" json:"status"`
	Protocol        ProtocolType   `gorm:"size:20;not null" json:"protocol"`
	ProfileID       *uuid.UUID     `gorm:"type:uuid;index" json:"profile_id,omitempty"`
	ParentID        *uuid.UUID     `gorm:"type:uuid;index" json:"parent_id,omitempty"`
	GatewayID       *uuid.UUID     `gorm:"type:uuid;index" json:"gateway_id,omitempty"`
	
	Labels          DeviceLabels   `gorm:"type:jsonb" json:"labels"`
	Tags            []string       `gorm:"type:jsonb" json:"tags"`
	Metadata        DeviceMetadata `gorm:"type:jsonb" json:"metadata"`
	
	ConnectionInfo  ConnectionInfo `gorm:"type:jsonb" json:"connection_info"`
	SecurityInfo    SecurityInfo   `gorm:"type:jsonb" json:"security_info"`
	
	LastOnlineAt    *time.Time     `json:"last_online_at,omitempty"`
	LastOfflineAt   *time.Time     `json:"last_offline_at,omitempty"`
	LastHeartbeatAt *time.Time     `json:"last_heartbeat_at,omitempty"`
	
	Enabled         bool           `gorm:"default:true" json:"enabled"`
	AutoRegister    bool           `gorm:"default:false" json:"auto_register"`
	
	Location        *DeviceLocation `gorm:"type:jsonb" json:"location,omitempty"`
}

type DeviceLabels map[string]string

type DeviceMetadata map[string]interface{}

type ConnectionInfo struct {
	Endpoint    string            `json:"endpoint,omitempty"`
	Port        int               `json:"port,omitempty"`
	BaudRate    int               `json:"baud_rate,omitempty"`
	DataBits    int               `json:"data_bits,omitempty"`
	StopBits    int               `json:"stop_bits,omitempty"`
	Parity      string            `json:"parity,omitempty"`
	SlaveID     int               `json:"slave_id,omitempty"`
	UnitID      int               `json:"unit_id,omitempty"`
	TopicPrefix string            `json:"topic_prefix,omitempty"`
	ClientID    string            `json:"client_id,omitempty"`
	KeepAlive   int               `json:"keep_alive,omitempty"`
	CleanSession bool             `json:"clean_session,omitempty"`
	QoS         int               `json:"qos,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

type SecurityInfo struct {
	AuthType       string `json:"auth_type,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"-"` // 不暴露密码
	Certificate    string `json:"-"` // 不暴露证书
	PrivateKey     string `json:"-"`
	APIKey         string `json:"-"`
	Token          string `json:"-"`
	TLSEnabled     bool   `json:"tls_enabled"`
	TLSSkipVerify  bool   `json:"tls_skip_verify"`
}

type DeviceLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude,omitempty"`
	Building  string  `json:"building,omitempty"`
	Floor     string  `json:"floor,omitempty"`
	Room      string  `json:"room,omitempty"`
	Zone      string  `json:"zone,omitempty"`
}

type DeviceProfile struct {
	IoTBaseModel
	TenantID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"tenant_id"`
	Name            string         `gorm:"size:100;not null;index" json:"name"`
	DisplayName     string         `gorm:"size:200" json:"display_name"`
	Description     string         `gorm:"type:text" json:"description"`
	Manufacturer    string         `gorm:"size:100" json:"manufacturer"`
	Model           string         `gorm:"size:100" json:"model"`
	Protocol        ProtocolType   `gorm:"size:20;not null" json:"protocol"`
	DeviceType      DeviceType     `gorm:"size:30" json:"device_type"`
	
	Properties      []DeviceProperty `gorm:"type:jsonb" json:"properties"`
	Commands        []DeviceCommand  `gorm:"type:jsonb" json:"commands"`
	Events          []DeviceEvent    `gorm:"type:jsonb" json:"events"`
	Alarms          []DeviceAlarm    `gorm:"type:jsonb" json:"alarms"`
	
	Labels          DeviceLabels   `gorm:"type:jsonb" json:"labels"`
}

type DeviceProperty struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	DataType    string `json:"data_type"`
	Unit        string `json:"unit,omitempty"`
	Min         interface{} `json:"min,omitempty"`
	Max         interface{} `json:"max,omitempty"`
	Default     interface{} `json:"default,omitempty"`
	AccessMode  string `json:"access_mode"` // r, w, rw
	Register    int    `json:"register,omitempty"`
	Address     string `json:"address,omitempty"`
	Scale       float64 `json:"scale,omitempty"`
	Offset      float64 `json:"offset,omitempty"`
	Required    bool   `json:"required"`
}

type DeviceCommand struct {
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Parameters  []CommandParameter     `json:"parameters,omitempty"`
	Returns     []CommandParameter     `json:"returns,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	Async       bool                   `json:"async"`
	Extra       map[string]interface{} `json:"extra,omitempty"`
}

type CommandParameter struct {
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	Description string      `json:"description"`
	DataType    string      `json:"data_type"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

type DeviceEventDefinition struct {
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Description string            `json:"description"`
	EventType   string            `json:"event_type"`
	Severity    string            `json:"severity"`
	Parameters  []EventParameter  `json:"parameters,omitempty"`
}

type EventParameter struct {
	Name        string      `json:"name"`
	DisplayName string      `json:"display_name"`
	DataType    string      `json:"data_type"`
}

type DeviceAlarm struct {
	Name         string      `json:"name"`
	DisplayName  string      `json:"display_name"`
	Description  string      `json:"description"`
	Condition    AlarmCondition `json:"condition"`
	Severity     string      `json:"severity"`
	Enabled      bool        `json:"enabled"`
}

type AlarmCondition struct {
	Property   string      `json:"property"`
	Operator   string      `json:"operator"`
	Threshold  interface{} `json:"threshold"`
	Duration   int         `json:"duration,omitempty"`
	Hysteresis float64     `json:"hysteresis,omitempty"`
}

type TelemetryData struct {
	ID          uuid.UUID              `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DeviceID    uuid.UUID              `gorm:"type:uuid;index;not null" json:"device_id"`
	TenantID    uuid.UUID              `gorm:"type:uuid;index;not null" json:"tenant_id"`
	Timestamp   time.Time              `gorm:"index;not null" json:"timestamp"`
	Property    string                 `gorm:"size:100;not null;index" json:"property"`
	Value       interface{}            `gorm:"type:jsonb" json:"value"`
	DataType    string                 `gorm:"size:20" json:"data_type"`
	Unit        string                 `gorm:"size:50" json:"unit,omitempty"`
	Quality     string                 `gorm:"size:20;default:'good'" json:"quality"`
	Metadata    map[string]interface{} `gorm:"type:jsonb" json:"metadata,omitempty"`
	
	CreatedAt   time.Time              `json:"created_at"`
}

type TelemetryBatch struct {
	DeviceID  uuid.UUID       `json:"device_id"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Timestamp time.Time       `json:"timestamp"`
	Values    []TelemetryItem `json:"values"`
}

type TelemetryItem struct {
	Property string      `json:"property"`
	Value    interface{} `json:"value"`
	DataType string      `json:"data_type,omitempty"`
	Unit     string      `json:"unit,omitempty"`
	Quality  string      `json:"quality,omitempty"`
}

type DeviceShadow struct {
	DeviceID    uuid.UUID              `json:"device_id"`
	Reported    ShadowState            `json:"reported"`
	Desired     ShadowState            `json:"desired"`
	Delta       ShadowState            `json:"delta"`
	Metadata    ShadowMetadata         `json:"metadata"`
	Version     int64                  `json:"version"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type ShadowState struct {
	Properties map[string]interface{} `json:"properties,omitempty"`
	Tags       map[string]string      `json:"tags,omitempty"`
}

type ShadowMetadata struct {
	Reported map[string]PropertyMetadata `json:"reported,omitempty"`
	Desired  map[string]PropertyMetadata `json:"desired,omitempty"`
}

type PropertyMetadata struct {
	Timestamp time.Time `json:"timestamp"`
	Version   int64     `json:"version"`
}

type DeviceCommandRequest struct {
	DeviceID    uuid.UUID              `json:"device_id"`
	CommandName string                 `json:"command_name"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	Async       bool                   `json:"async"`
	CorrelationID string               `json:"correlation_id,omitempty"`
}

type DeviceCommandResponse struct {
	DeviceID      uuid.UUID              `json:"device_id"`
	CommandName   string                 `json:"command_name"`
	CorrelationID string                 `json:"correlation_id"`
	Status        string                 `json:"status"`
	Result        map[string]interface{} `json:"result,omitempty"`
	ErrorCode     string                 `json:"error_code,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
	Timestamp     time.Time              `json:"timestamp"`
}

type DeviceEventRecord struct {
	IoTBaseModel
	DeviceID    uuid.UUID              `gorm:"type:uuid;index;not null" json:"device_id"`
	TenantID    uuid.UUID              `gorm:"type:uuid;index;not null" json:"tenant_id"`
	EventName   string                 `gorm:"size:100;not null;index" json:"event_name"`
	EventType   string                 `gorm:"size:30" json:"event_type"`
	Severity    string                 `gorm:"size:20" json:"severity"`
	Payload     map[string]interface{} `gorm:"type:jsonb" json:"payload"`
	Processed   bool                   `gorm:"default:false" json:"processed"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
}

type DeviceAlarmRecord struct {
	IoTBaseModel
	DeviceID      uuid.UUID              `gorm:"type:uuid;index;not null" json:"device_id"`
	TenantID      uuid.UUID              `gorm:"type:uuid;index;not null" json:"tenant_id"`
	AlarmName     string                 `gorm:"size:100;not null;index" json:"alarm_name"`
	AlarmType     string                 `gorm:"size:30" json:"alarm_type"`
	Severity      string                 `gorm:"size:20" json:"severity"`
	Status        string                 `gorm:"size:20;default:'active';index" json:"status"`
	TriggeredAt   time.Time              `json:"triggered_at"`
	ClearedAt     *time.Time             `json:"cleared_at,omitempty"`
	AcknowledgedAt *time.Time            `json:"acknowledged_at,omitempty"`
	AcknowledgedBy *uuid.UUID            `json:"acknowledged_by,omitempty"`
	Message       string                 `gorm:"type:text" json:"message"`
	Details       map[string]interface{} `gorm:"type:jsonb" json:"details"`
}

type EnergyDevice struct {
	IoTBaseModel
	DeviceID        uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"device_id"`
	TenantID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"tenant_id"`
	DeviceType      string         `gorm:"size:30;not null" json:"device_type"`
	Capacity        float64        `json:"capacity"`
	Unit            string         `gorm:"size:20" json:"unit"`
	
	CurrentPower    float64        `json:"current_power"`
	TotalEnergy     float64        `json:"total_energy"`
	PeakPower       float64        `json:"peak_power"`
	MinPower        float64        `json:"min_power"`
	AvgPower        float64        `json:"avg_power"`
	
	PowerFactor     float64        `json:"power_factor"`
	Voltage         float64        `json:"voltage"`
	Current         float64        `json:"current"`
	Frequency       float64        `json:"frequency"`
	
	Status          string         `gorm:"size:20" json:"status"`
	LastReadingAt   *time.Time     `json:"last_reading_at,omitempty"`
}

type ComputeDevice struct {
	IoTBaseModel
	DeviceID        uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"device_id"`
	TenantID        uuid.UUID      `gorm:"type:uuid;index;not null" json:"tenant_id"`
	DeviceType      string         `gorm:"size:30;not null" json:"device_type"`
	
	CPUModel        string         `gorm:"size:100" json:"cpu_model"`
	CPUCores        int            `json:"cpu_cores"`
	CPUUsage        float64        `json:"cpu_usage"`
	
	MemoryTotal     int64          `json:"memory_total"`
	MemoryUsed      int64          `json:"memory_used"`
	MemoryUsage     float64        `json:"memory_usage"`
	
	GPUCount        int            `json:"gpu_count"`
	GPUModel        string         `gorm:"size:100" json:"gpu_model,omitempty"`
	GPUUsage        float64        `json:"gpu_usage"`
	GPUMemoryUsed   int64          `json:"gpu_memory_used"`
	GPUMemoryTotal  int64          `json:"gpu_memory_total"`
	
	DiskTotal       int64          `json:"disk_total"`
	DiskUsed        int64          `json:"disk_used"`
	DiskUsage       float64        `json:"disk_usage"`
	
	NetworkIn       int64          `json:"network_in"`
	NetworkOut      int64          `json:"network_out"`
	
	Temperature     float64        `json:"temperature"`
	PowerConsumption float64       `json:"power_consumption"`
	
	Status          string         `gorm:"size:20" json:"status"`
	LastReadingAt   *time.Time     `json:"last_reading_at,omitempty"`
}

type ProtocolConfig struct {
	Protocol    ProtocolType       `json:"protocol"`
	Enabled     bool               `json:"enabled"`
	Config      interface{}        `json:"config,omitempty"`
}

type MQTTConfig struct {
	Broker          string            `json:"broker"`
	Port            int               `json:"port"`
	ClientID        string            `json:"client_id,omitempty"`
	Username        string            `json:"username,omitempty"`
	Password        string            `json:"-"`
	CleanSession    bool              `json:"clean_session"`
	KeepAlive       int               `json:"keep_alive"`
	QoS             int               `json:"qos"`
	TopicPrefix     string            `json:"topic_prefix"`
	TLSEnabled      bool              `json:"tls_enabled"`
	TLSSkipVerify   bool              `json:"tls_skip_verify"`
	Certificate     string            `json:"-"`
	PrivateKey      string            `json:"-"`
	AutoReconnect   bool              `json:"auto_reconnect"`
	MaxReconnect    int               `json:"max_reconnect"`
	ReconnectDelay  int               `json:"reconnect_delay"`
	WillTopic       string            `json:"will_topic,omitempty"`
	WillPayload     string            `json:"will_payload,omitempty"`
	WillQoS         int               `json:"will_qos"`
	WillRetained    bool              `json:"will_retained"`
}

type ModbusConfig struct {
	Protocol    string `json:"protocol"` // tcp, rtu
	Host        string `json:"host,omitempty"`
	Port        int    `json:"port,omitempty"`
	SerialPort  string `json:"serial_port,omitempty"`
	BaudRate    int    `json:"baud_rate,omitempty"`
	DataBits    int    `json:"data_bits,omitempty"`
	StopBits    int    `json:"stop_bits,omitempty"`
	Parity      string `json:"parity,omitempty"`
	SlaveID     int    `json:"slave_id"`
	Timeout     int    `json:"timeout"`
	PollingRate int    `json:"polling_rate"`
}

type OPCUAConfig struct {
	Endpoint        string            `json:"endpoint"`
	SecurityPolicy  string            `json:"security_policy"`
	SecurityMode    string            `json:"security_mode"`
	AuthType        string            `json:"auth_type"`
	Username        string            `json:"username,omitempty"`
	Password        string            `json:"-"`
	Certificate     string            `json:"-"`
	PrivateKey      string            `json:"-"`
	SessionTimeout  int               `json:"session_timeout"`
	RequestTimeout  int               `json:"request_timeout"`
	NodeIDs         []string          `json:"node_ids,omitempty"`
	PollingRate     int               `json:"polling_rate"`
}

type ConnectorStatus struct {
	Protocol      ProtocolType `json:"protocol"`
	Status        string       `json:"status"`
	ConnectedAt   *time.Time   `json:"connected_at,omitempty"`
	DisconnectedAt *time.Time  `json:"disconnected_at,omitempty"`
	ReconnectCount int         `json:"reconnect_count"`
	LastError     string       `json:"last_error,omitempty"`
	DevicesOnline int          `json:"devices_online"`
	DevicesTotal  int          `json:"devices_total"`
	MessagesIn    int64        `json:"messages_in"`
	MessagesOut   int64        `json:"messages_out"`
	BytesIn       int64        `json:"bytes_in"`
	BytesOut      int64        `json:"bytes_out"`
}

type TelemetryQuery struct {
	DeviceIDs  []uuid.UUID `json:"device_ids,omitempty"`
	Properties []string    `json:"properties,omitempty"`
	StartTime  *time.Time  `json:"start_time,omitempty"`
	EndTime    *time.Time  `json:"end_time,omitempty"`
	Limit      int         `json:"limit,omitempty"`
	Offset     int         `json:"offset,omitempty"`
	Order      string      `json:"order,omitempty"` // asc, desc
	Aggregate  string      `json:"aggregate,omitempty"` // avg, min, max, sum
	Interval   int         `json:"interval,omitempty"` // seconds
}

type TelemetryQueryResult struct {
	DeviceID  uuid.UUID       `json:"device_id"`
	Property  string          `json:"property"`
	Points    []TelemetryPoint `json:"points"`
}

type TelemetryPoint struct {
	Timestamp time.Time      `json:"timestamp"`
	Value     interface{}    `json:"value"`
	Quality   string         `json:"quality,omitempty"`
}
