package protocol

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ModbusFunctionCode byte

const (
	ModbusReadCoils            ModbusFunctionCode = 0x01
	ModbusReadDiscreteInputs   ModbusFunctionCode = 0x02
	ModbusReadHoldingRegisters ModbusFunctionCode = 0x03
	ModbusReadInputRegisters   ModbusFunctionCode = 0x04
	ModbusWriteSingleCoil      ModbusFunctionCode = 0x05
	ModbusWriteSingleRegister  ModbusFunctionCode = 0x06
	ModbusWriteMultipleCoils   ModbusFunctionCode = 0x0F
	ModbusWriteMultipleRegisters ModbusFunctionCode = 0x10
)

type ModbusRegisterType string

const (
	ModbusRegisterCoil            ModbusRegisterType = "coil"
	ModbusRegisterDiscreteInput   ModbusRegisterType = "discrete_input"
	ModbusRegisterHoldingRegister ModbusRegisterType = "holding_register"
	ModbusRegisterInputRegister   ModbusRegisterType = "input_register"
)

type ModbusRegister struct {
	Name      string             `json:"name"`
	Address   uint16             `json:"address"`
	Quantity  uint16             `json:"quantity"`
	Type      ModbusRegisterType `json:"type"`
	DataType  string             `json:"data_type"`
	Scale     float64            `json:"scale"`
	Offset    float64            `json:"offset"`
	ByteOrder string             `json:"byte_order"`
}

type ModbusConfig struct {
	Protocol    string `json:"protocol"`
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

type ModbusDeviceConfig struct {
	DeviceID  uuid.UUID
	SlaveID   byte
	Registers []ModbusRegister
	PollRate  int
}

type ModbusAdapter struct {
	config  *ModbusConfig
	handler MessageHandler

	tcpConn net.Conn
	rtuPort ReadWriteCloser

	connected bool
	connectMu sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup

	deviceConfigs map[uuid.UUID]*ModbusDeviceConfig
	deviceMu      sync.RWMutex

	pollingTasks map[uuid.UUID]context.CancelFunc
	pollingMu    sync.RWMutex
}

type ReadWriteCloser interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
}

func NewModbusAdapter(config *ModbusConfig) *ModbusAdapter {
	return &ModbusAdapter{
		config:        config,
		stopCh:        make(chan struct{}),
		deviceConfigs: make(map[uuid.UUID]*ModbusDeviceConfig),
		pollingTasks:  make(map[uuid.UUID]context.CancelFunc),
	}
}

func (a *ModbusAdapter) Start(ctx context.Context) error {
	var err error

	switch a.config.Protocol {
	case "tcp":
		err = a.connectTCP(ctx)
	case "rtu":
		err = a.connectRTU(ctx)
	default:
		err = fmt.Errorf("不支持的Modbus协议: %s", a.config.Protocol)
	}

	if err != nil {
		return err
	}

	a.setConnected(true)
	log.Printf("[ModbusAdapter] Modbus %s 连接成功", a.config.Protocol)

	return nil
}

func (a *ModbusAdapter) connectTCP(ctx context.Context) error {
	address := fmt.Sprintf("%s:%d", a.config.Host, a.config.Port)

	dialer := &net.Dialer{
		Timeout: time.Duration(a.config.Timeout) * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("连接Modbus TCP失败: %w", err)
	}

	a.tcpConn = conn
	return nil
}

func (a *ModbusAdapter) connectRTU(ctx context.Context) error {
	return fmt.Errorf("Modbus RTU需要串口库支持，请安装相关依赖")
}

func (a *ModbusAdapter) Stop() error {
	close(a.stopCh)

	a.pollingMu.Lock()
	for deviceID, cancel := range a.pollingTasks {
		cancel()
		delete(a.pollingTasks, deviceID)
	}
	a.pollingMu.Unlock()

	a.wg.Wait()

	if a.tcpConn != nil {
		a.tcpConn.Close()
	}

	a.setConnected(false)
	log.Println("[ModbusAdapter] Modbus连接已关闭")
	return nil
}

func (a *ModbusAdapter) Reconnect() error {
	if a.tcpConn != nil {
		a.tcpConn.Close()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.Start(ctx)
}

func (a *ModbusAdapter) IsConnected() bool {
	a.connectMu.RLock()
	defer a.connectMu.RUnlock()
	return a.connected
}

func (a *ModbusAdapter) SetMessageHandler(handler MessageHandler) {
	a.handler = handler
}

func (a *ModbusAdapter) Subscribe(topic string) error {
	return nil
}

func (a *ModbusAdapter) Unsubscribe(topic string) error {
	return nil
}

func (a *ModbusAdapter) Publish(topic string, payload []byte) error {
	return nil
}

func (a *ModbusAdapter) SendCommand(deviceID uuid.UUID, connectionInfo interface{}, cmd *DeviceCommandRequest) (map[string]interface{}, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("Modbus未连接")
	}

	var slaveID byte = byte(a.config.SlaveID)
	if ci, ok := connectionInfo.(*DeviceConnectionInfo); ok && ci.SlaveID > 0 {
		slaveID = byte(ci.SlaveID)
	}

	switch cmd.CommandName {
	case "write_coil":
		return a.writeCoil(slaveID, cmd.Parameters)
	case "write_register":
		return a.writeRegister(slaveID, cmd.Parameters)
	case "write_multiple_registers":
		return a.writeMultipleRegisters(slaveID, cmd.Parameters)
	default:
		return nil, fmt.Errorf("不支持的命令: %s", cmd.CommandName)
	}
}

func (a *ModbusAdapter) RegisterDevice(deviceID uuid.UUID, slaveID byte, registers []ModbusRegister, pollRate int) {
	a.deviceMu.Lock()
	defer a.deviceMu.Unlock()

	a.deviceConfigs[deviceID] = &ModbusDeviceConfig{
		DeviceID:  deviceID,
		SlaveID:   slaveID,
		Registers: registers,
		PollRate:  pollRate,
	}

	if pollRate > 0 {
		go a.startPolling(deviceID)
	}

	log.Printf("[ModbusAdapter] 已注册设备 %s，从站ID: %d，寄存器数量: %d", deviceID, slaveID, len(registers))
}

func (a *ModbusAdapter) UnregisterDevice(deviceID uuid.UUID) {
	a.deviceMu.Lock()
	delete(a.deviceConfigs, deviceID)
	a.deviceMu.Unlock()

	a.pollingMu.Lock()
	if cancel, ok := a.pollingTasks[deviceID]; ok {
		cancel()
		delete(a.pollingTasks, deviceID)
	}
	a.pollingMu.Unlock()

	log.Printf("[ModbusAdapter] 已注销设备 %s", deviceID)
}

func (a *ModbusAdapter) startPolling(deviceID uuid.UUID) {
	a.deviceMu.RLock()
	config, ok := a.deviceConfigs[deviceID]
	if !ok {
		a.deviceMu.RUnlock()
		return
	}
	pollRate := config.PollRate
	a.deviceMu.RUnlock()

	ctx, cancel := context.WithCancel(context.Background())

	a.pollingMu.Lock()
	a.pollingTasks[deviceID] = cancel
	a.pollingMu.Unlock()

	ticker := time.NewTicker(time.Duration(pollRate) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.pollDevice(deviceID)
		}
	}
}

func (a *ModbusAdapter) pollDevice(deviceID uuid.UUID) {
	a.deviceMu.RLock()
	config, ok := a.deviceConfigs[deviceID]
	if !ok {
		a.deviceMu.RUnlock()
		return
	}
	registers := config.Registers
	slaveID := config.SlaveID
	a.deviceMu.RUnlock()

	data := make(map[string]interface{})

	for _, reg := range registers {
		value, err := a.readRegister(slaveID, reg)
		if err != nil {
			log.Printf("[ModbusAdapter] 读取寄存器 %s 失败: %v", reg.Name, err)
			continue
		}
		data[reg.Name] = value
	}

	if a.handler != nil && len(data) > 0 {
		payload := []byte(fmt.Sprintf("%v", data))
		a.handler(fmt.Sprintf("modbus/%s/data", deviceID), payload)
	}
}

func (a *ModbusAdapter) readRegister(slaveID byte, reg ModbusRegister) (interface{}, error) {
	var functionCode ModbusFunctionCode
	var quantity uint16 = reg.Quantity

	switch reg.Type {
	case ModbusRegisterCoil:
		functionCode = ModbusReadCoils
		if quantity == 0 {
			quantity = 1
		}
	case ModbusRegisterDiscreteInput:
		functionCode = ModbusReadDiscreteInputs
		if quantity == 0 {
			quantity = 1
		}
	case ModbusRegisterHoldingRegister:
		functionCode = ModbusReadHoldingRegisters
		if quantity == 0 {
			quantity = 1
		}
	case ModbusRegisterInputRegister:
		functionCode = ModbusReadInputRegisters
		if quantity == 0 {
			quantity = 1
		}
	default:
		return nil, fmt.Errorf("不支持的寄存器类型: %s", reg.Type)
	}

	response, err := a.sendModbusTCP(slaveID, byte(functionCode), reg.Address, quantity)
	if err != nil {
		return nil, err
	}

	return a.parseRegisterValue(response, reg)
}

func (a *ModbusAdapter) sendModbusTCP(slaveID, functionCode byte, address uint16, quantity uint16) ([]byte, error) {
	if a.tcpConn == nil {
		return nil, fmt.Errorf("TCP连接未建立")
	}

	transactionID := uint16(time.Now().UnixNano() & 0xFFFF)

	request := make([]byte, 12)
	binary.BigEndian.PutUint16(request[0:2], transactionID)
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], 6)
	request[6] = slaveID
	request[7] = functionCode
	binary.BigEndian.PutUint16(request[8:10], address)
	binary.BigEndian.PutUint16(request[10:12], quantity)

	if _, err := a.tcpConn.Write(request); err != nil {
		return nil, fmt.Errorf("发送Modbus请求失败: %w", err)
	}

	responseHeader := make([]byte, 8)
	if _, err := a.tcpConn.Read(responseHeader); err != nil {
		return nil, fmt.Errorf("读取Modbus响应头失败: %w", err)
	}

	respTransactionID := binary.BigEndian.Uint16(responseHeader[0:2])
	if respTransactionID != transactionID {
		return nil, fmt.Errorf("事务ID不匹配")
	}

	length := binary.BigEndian.Uint16(responseHeader[4:6])
	responseData := make([]byte, length)
	if _, err := a.tcpConn.Read(responseData); err != nil {
		return nil, fmt.Errorf("读取Modbus响应数据失败: %w", err)
	}

	if responseData[1]&0x80 != 0 {
		return nil, fmt.Errorf("Modbus异常响应: %d", responseData[2])
	}

	return responseData, nil
}

func (a *ModbusAdapter) parseRegisterValue(response []byte, reg ModbusRegister) (interface{}, error) {
	if len(response) < 3 {
		return nil, fmt.Errorf("响应数据长度不足")
	}

	byteCount := int(response[2])
	if len(response) < 3+byteCount {
		return nil, fmt.Errorf("响应数据不完整")
	}

	data := response[3 : 3+byteCount]

	switch reg.Type {
	case ModbusRegisterCoil, ModbusRegisterDiscreteInput:
		return data[0] != 0, nil

	case ModbusRegisterHoldingRegister, ModbusRegisterInputRegister:
		switch reg.DataType {
		case "uint16":
			value := binary.BigEndian.Uint16(data[0:2])
			return a.applyScaleAndOffset(float64(value), reg), nil

		case "int16":
			value := int16(binary.BigEndian.Uint16(data[0:2]))
			return a.applyScaleAndOffset(float64(value), reg), nil

		case "uint32":
			var value uint32
			if reg.ByteOrder == "little" {
				value = binary.LittleEndian.Uint32(data[0:4])
			} else {
				value = binary.BigEndian.Uint32(data[0:4])
			}
			return a.applyScaleAndOffset(float64(value), reg), nil

		case "int32":
			var value int32
			if reg.ByteOrder == "little" {
				value = int32(binary.LittleEndian.Uint32(data[0:4]))
			} else {
				value = int32(binary.BigEndian.Uint32(data[0:4]))
			}
			return a.applyScaleAndOffset(float64(value), reg), nil

		case "float32":
			bits := binary.BigEndian.Uint32(data[0:4])
			value := float32frombits(bits)
			return a.applyScaleAndOffset(float64(value), reg), nil

		default:
			value := binary.BigEndian.Uint16(data[0:2])
			return a.applyScaleAndOffset(float64(value), reg), nil
		}
	}

	return nil, fmt.Errorf("无法解析寄存器值")
}

func (a *ModbusAdapter) applyScaleAndOffset(value float64, reg ModbusRegister) float64 {
	if reg.Scale != 0 {
		value = value * reg.Scale
	}
	value = value + reg.Offset
	return value
}

func (a *ModbusAdapter) writeCoil(slaveID byte, params map[string]interface{}) (map[string]interface{}, error) {
	address, ok := params["address"].(float64)
	if !ok {
		return nil, fmt.Errorf("缺少address参数")
	}

	value, ok := params["value"].(bool)
	if !ok {
		return nil, fmt.Errorf("缺少value参数")
	}

	var coilValue uint16 = 0x0000
	if value {
		coilValue = 0xFF00
	}

	request := make([]byte, 12)
	binary.BigEndian.PutUint16(request[0:2], uint16(time.Now().UnixNano()&0xFFFF))
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], 6)
	request[6] = slaveID
	request[7] = byte(ModbusWriteSingleCoil)
	binary.BigEndian.PutUint16(request[8:10], uint16(address))
	binary.BigEndian.PutUint16(request[10:12], coilValue)

	if _, err := a.tcpConn.Write(request); err != nil {
		return nil, fmt.Errorf("写入线圈失败: %w", err)
	}

	response := make([]byte, 12)
	if _, err := a.tcpConn.Read(response); err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"address": address,
		"value":   value,
	}, nil
}

func (a *ModbusAdapter) writeRegister(slaveID byte, params map[string]interface{}) (map[string]interface{}, error) {
	address, ok := params["address"].(float64)
	if !ok {
		return nil, fmt.Errorf("缺少address参数")
	}

	value, ok := params["value"].(float64)
	if !ok {
		return nil, fmt.Errorf("缺少value参数")
	}

	request := make([]byte, 12)
	binary.BigEndian.PutUint16(request[0:2], uint16(time.Now().UnixNano()&0xFFFF))
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], 6)
	request[6] = slaveID
	request[7] = byte(ModbusWriteSingleRegister)
	binary.BigEndian.PutUint16(request[8:10], uint16(address))
	binary.BigEndian.PutUint16(request[10:12], uint16(value))

	if _, err := a.tcpConn.Write(request); err != nil {
		return nil, fmt.Errorf("写入寄存器失败: %w", err)
	}

	response := make([]byte, 12)
	if _, err := a.tcpConn.Read(response); err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"address": address,
		"value":   value,
	}, nil
}

func (a *ModbusAdapter) writeMultipleRegisters(slaveID byte, params map[string]interface{}) (map[string]interface{}, error) {
	address, ok := params["address"].(float64)
	if !ok {
		return nil, fmt.Errorf("缺少address参数")
	}

	values, ok := params["values"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("缺少values参数")
	}

	quantity := len(values)
	byteCount := quantity * 2

	requestLen := 13 + byteCount
	request := make([]byte, requestLen)
	binary.BigEndian.PutUint16(request[0:2], uint16(time.Now().UnixNano()&0xFFFF))
	binary.BigEndian.PutUint16(request[2:4], 0)
	binary.BigEndian.PutUint16(request[4:6], uint16(7+byteCount))
	request[6] = slaveID
	request[7] = byte(ModbusWriteMultipleRegisters)
	binary.BigEndian.PutUint16(request[8:10], uint16(address))
	binary.BigEndian.PutUint16(request[10:12], uint16(quantity))
	request[12] = byte(byteCount)

	for i, v := range values {
		val, _ := v.(float64)
		binary.BigEndian.PutUint16(request[13+i*2:15+i*2], uint16(val))
	}

	if _, err := a.tcpConn.Write(request); err != nil {
		return nil, fmt.Errorf("写入多个寄存器失败: %w", err)
	}

	response := make([]byte, 12)
	if _, err := a.tcpConn.Read(response); err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	return map[string]interface{}{
		"success":  true,
		"address":  address,
		"quantity": quantity,
	}, nil
}

func (a *ModbusAdapter) setConnected(connected bool) {
	a.connectMu.Lock()
	defer a.connectMu.Unlock()
	a.connected = connected
}

func (a *ModbusAdapter) ReadCoils(slaveID byte, address uint16, quantity uint16) ([]bool, error) {
	response, err := a.sendModbusTCP(slaveID, byte(ModbusReadCoils), address, quantity)
	if err != nil {
		return nil, err
	}

	byteCount := int(response[2])
	data := response[3 : 3+byteCount]

	coils := make([]bool, quantity)
	for i := uint16(0); i < quantity; i++ {
		byteIndex := i / 8
		bitIndex := i % 8
		coils[i] = (data[byteIndex] & (1 << bitIndex)) != 0
	}

	return coils, nil
}

func (a *ModbusAdapter) ReadHoldingRegisters(slaveID byte, address uint16, quantity uint16) ([]uint16, error) {
	response, err := a.sendModbusTCP(slaveID, byte(ModbusReadHoldingRegisters), address, quantity)
	if err != nil {
		return nil, err
	}

	byteCount := int(response[2])
	data := response[3 : 3+byteCount]

	registers := make([]uint16, quantity)
	for i := uint16(0); i < quantity; i++ {
		registers[i] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
	}

	return registers, nil
}

func (a *ModbusAdapter) ReadInputRegisters(slaveID byte, address uint16, quantity uint16) ([]uint16, error) {
	response, err := a.sendModbusTCP(slaveID, byte(ModbusReadInputRegisters), address, quantity)
	if err != nil {
		return nil, err
	}

	byteCount := int(response[2])
	data := response[3 : 3+byteCount]

	registers := make([]uint16, quantity)
	for i := uint16(0); i < quantity; i++ {
		registers[i] = binary.BigEndian.Uint16(data[i*2 : i*2+2])
	}

	return registers, nil
}

func float32frombits(b uint32) float32 {
	return float32(float64(b))
}
