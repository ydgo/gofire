package event

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestEventClone(t *testing.T) {
	original := NewEvent()
	defer original.Release()

	// 测试基本类型
	original.AddField("string", "test")
	original.AddField("int", 123)
	original.AddField("float", 123.45)
	original.AddField("bool", true)

	// 测试切片
	original.AddField("slice", []interface{}{"a", 1, true})
	original.AddField("intSlice", []int{1, 2, 3})

	// 测试嵌套 map
	original.AddField("map", map[string]interface{}{
		"nested": map[string]interface{}{
			"key":  "value",
			"nums": []int{1, 2, 3},
		},
	})

	// 测试时间
	original.AddField("time", time.Now())

	// 克隆
	cloned := original.Clone()
	defer cloned.Release()

	// 修改克隆后的值，确保不影响原值
	if m, ok := cloned.Fields["map"].(map[string]interface{}); ok {
		if nested, ok := m["nested"].(map[string]interface{}); ok {
			nested["key"] = "modified"
		}
	}
	if s, ok := cloned.Fields["slice"].([]interface{}); ok {
		s[0] = "modified"
	}

	// 验证原值未被修改
	if m, ok := original.Fields["map"].(map[string]interface{}); ok {
		if nested, ok := m["nested"].(map[string]interface{}); ok {
			if nested["key"] != "value" {
				t.Error("Original map was modified")
			}
		}
	}
	if s, ok := original.Fields["slice"].([]interface{}); ok {
		if s[0] != "a" {
			t.Error("Original slice was modified")
		}
	}
}

func TestEventJSONMarshalUnmarshal(t *testing.T) {
	// 创建一个固定时间的事件
	evt := &Event{
		Timestamp: time.Date(2024, 3, 14, 15, 30, 0, 0, time.UTC),
		Message:   "test message",
		Fields:    make(map[string]interface{}),
	}
	evt.Fields["string"] = "value"
	evt.Fields["number"] = float64(123) // 明确使用 float64 类型

	// 序列化
	data, err := json.Marshal(evt)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// 反序列化到新的事件
	restored := &Event{
		Fields: make(map[string]interface{}),
	}

	if err := json.Unmarshal(data, restored); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// 逐个比较字段
	if restored.Message != evt.Message {
		t.Errorf("Message mismatch: got %q, want %q", restored.Message, evt.Message)
	}

	if !restored.Timestamp.Equal(evt.Timestamp) {
		t.Errorf("Timestamp mismatch: got %v, want %v", restored.Timestamp, evt.Timestamp)
	}

	// 比较 Fields
	if len(restored.Fields) != len(evt.Fields) {
		t.Errorf("Fields length mismatch: got %d, want %d", len(restored.Fields), len(evt.Fields))
	}

	for k, v := range evt.Fields {
		rv, ok := restored.Fields[k]
		if !ok {
			t.Errorf("Missing field %q in restored event", k)
			continue
		}
		if !reflect.DeepEqual(rv, v) {
			t.Errorf("Field %q mismatch: got %#v (%T), want %#v (%T)",
				k, rv, rv, v, v)
		}
	}

	// 输出实际的 JSON 以便调试
	t.Logf("Generated JSON: %s", string(data))
}

func TestToJSON(t *testing.T) {
	evt := &Event{
		Timestamp: time.Date(2024, 3, 14, 15, 30, 0, 0, time.UTC),
		Message:   "test message",
		Fields:    make(map[string]interface{}),
	}
	evt.Fields["string"] = "value"
	evt.Fields["number"] = float64(123) // 明确使用 float64 类型
	defer evt.Release()
	data, err := evt.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}
	t.Logf("ToJSON: %s\n", data)
}
