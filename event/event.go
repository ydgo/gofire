package event

import (
	"encoding/json"
	"sync"
	"time"
)

// Event 表示一条日志事件
type Event struct {
	Timestamp time.Time              // 事件发生时间
	Fields    map[string]interface{} // 事件字段
	Message   string                 // 原始消息
}

// 创建一个全局的对象池
var eventPool = &sync.Pool{
	New: func() interface{} {
		return &Event{
			Fields: make(map[string]interface{}, 8), // 预分配一个合理大小的 map
		}
	},
}

// NewEvent 从对象池获取一个新的 Event
func NewEvent() *Event {
	event := eventPool.Get().(*Event)
	event.Timestamp = time.Now()
	return event
}

// Release 将 Event 放回对象池以供复用
func (e *Event) Release() {
	// 清理字段
	for k := range e.Fields {
		delete(e.Fields, k)
	}
	e.Message = ""
	eventPool.Put(e)
}

// AddField 添加一个字段
func (e *Event) AddField(key string, value interface{}) {
	e.Fields[key] = value
}

// GetField 获取字段值
func (e *Event) GetField(key string) (interface{}, bool) {
	val, ok := e.Fields[key]
	return val, ok
}

// SetMessage 设置原始消息
func (e *Event) SetMessage(msg string) {
	e.Message = msg
}

func (e *Event) GetString(key string) (string, bool) {
	if val, ok := e.Fields[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

func (e *Event) GetInt(key string) (int, bool) {
	if val, ok := e.Fields[key]; ok {
		switch v := val.(type) {
		case int:
			return v, true
		case float64:
			return int(v), true
		}
	}
	return 0, false
}

func (e *Event) GetFloat(key string) (float64, bool) {
	if val, ok := e.Fields[key]; ok {
		switch v := val.(type) {
		case float64:
			return v, true
		case int:
			return float64(v), true
		}
	}
	return 0.0, false
}

func (e *Event) GetBool(key string) (bool, bool) {
	if val, ok := e.Fields[key]; ok {
		if b, ok := val.(bool); ok {
			return b, true
		}
	}
	return false, false
}

func (e *Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Timestamp string                 `json:"timestamp"`
		Fields    map[string]interface{} `json:"fields"`
		Message   string                 `json:"message"`
	}{
		Timestamp: e.Timestamp.Format(time.RFC3339),
		Fields:    e.Fields,
		Message:   e.Message,
	})
}

func (e *Event) UnmarshalJSON(data []byte) error {
	// 定义一个临时结构体来接收数据
	aux := struct {
		Timestamp string                 `json:"timestamp"`
		Fields    map[string]interface{} `json:"fields"`
		Message   string                 `json:"message"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 解析时间
	timestamp, err := time.Parse(time.RFC3339, aux.Timestamp)
	if err != nil {
		return err
	}

	// 填充 Event 字段
	e.Timestamp = timestamp
	e.Fields = aux.Fields
	e.Message = aux.Message

	return nil
}

func (e *Event) ToJSON() (string, error) {
	bytes, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Clone 创建 Event 的深拷贝
func (e *Event) Clone() *Event {
	newEvent := NewEvent()

	// 复制时间戳
	newEvent.Timestamp = e.Timestamp

	// 复制消息
	newEvent.Message = e.Message

	// 深复制字段
	for k, v := range e.Fields {
		newEvent.Fields[k] = deepCopy(v)
	}

	return newEvent
}

// deepCopy 处理值的深拷贝
func deepCopy(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []interface{}:
		newSlice := make([]interface{}, len(val))
		for i, item := range val {
			newSlice[i] = deepCopy(item)
		}
		return newSlice

	case map[string]interface{}:
		newMap := make(map[string]interface{}, len(val))
		for k, v := range val {
			newMap[k] = deepCopy(v)
		}
		return newMap

	case map[interface{}]interface{}:
		newMap := make(map[interface{}]interface{}, len(val))
		for k, v := range val {
			newMap[deepCopy(k)] = deepCopy(v)
		}
		return newMap

	case time.Time:
		return val

	case *time.Time:
		if val == nil {
			return nil
		}
		newTime := *val
		return &newTime

	// 处理基本类型
	case string, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64, bool:
		return val

	// 处理切片类型
	case []string:
		newSlice := make([]string, len(val))
		copy(newSlice, val)
		return newSlice

	case []int:
		newSlice := make([]int, len(val))
		copy(newSlice, val)
		return newSlice

	case []float64:
		newSlice := make([]float64, len(val))
		copy(newSlice, val)
		return newSlice

	default:
		// 对于不能确定如何复制的类型，可以：
		// 1. 尝试 JSON 序列化然后反序列化
		// 2. 或者返回原值（浅拷贝）
		// 3. 或者返回 nil
		// 这里选择方案1，最安全但性能较低
		if jsonBytes, err := json.Marshal(val); err == nil {
			var newVal interface{}
			if err := json.Unmarshal(jsonBytes, &newVal); err == nil {
				return newVal
			}
		}
		// 如果 JSON 序列化失败，返回 nil
		return nil
	}
}

// 添加测试用例
