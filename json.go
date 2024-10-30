package gosonify

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type Json struct {
	value any
}

type token struct {
	start rune
	end   rune
	depth int
	sep   rune
}

type (
	stateMap map[rune]*token
	Object   map[string]any
)
type Gosonified interface {
	Get(holder any, path string) error
}

func (o Object) Get(holder any, path string) error {
	err := getHelper(o, holder, path)
	if err != nil {
		return err
	}
	return nil
}

func getHelper(jsonData any, holder any, path string) error {
	if len(path) == 0 {
		switch h := holder.(type) {
		case *[]any:
			jsonArray, ok := jsonData.([]any)
			if !ok {
				return errors.New("mismatched types" + reflect.TypeOf(jsonData).String() + reflect.TypeOf(holder).String())
			}
			*h = jsonArray
		case *Object:
			jsonObject, ok := jsonData.(Object)
			if !ok {
				return errors.New("mismatched types" + reflect.TypeOf(jsonData).String() + reflect.TypeOf(holder).String())
			}
			*h = jsonObject
		case *[]Object:
			jsonArray, ok := jsonData.([]any)
			if !ok {
				return errors.New("mismatched types" + reflect.TypeOf(jsonData).String() + reflect.TypeOf(holder).String())
			}
			var jsonArrayOfObject []Object
			for _, arrayElement := range jsonArray {
				object, ok := arrayElement.(Object)
				if !ok {
					return errors.New("mismatched types" + reflect.TypeOf(jsonData).String() + reflect.TypeOf(holder).String())
				}
				jsonArrayOfObject = append(jsonArrayOfObject, object)
			}
			*h = jsonArrayOfObject
		default:
			return errors.New("mismatched types" + reflect.TypeOf(jsonData).String() + reflect.TypeOf(holder).String())
		}
		return nil
	}
	if path[0] == '.' {
		path = path[1:]
	}
	if path[0] != '[' {
		object, ok := jsonData.(Object)
		if !ok {
			return errors.New("err getting")
		}
		objectIndex := strings.Split(path, "[")[0]
		err := getHelper(object[objectIndex], holder, path[len(objectIndex):])
		if err != nil {
			return err
		}
	} else {
		array, ok := jsonData.([]any)
		if !ok {
			return errors.New("err getting")
		}
		arrayIndexStart := strings.Split(path, "[")[1]
		arrayIndex := strings.Split(arrayIndexStart, "]")[0]
		nextPath := path[len(path)-(len(path)-1-len(arrayIndex)-1):]
		num, err := strconv.Atoi(arrayIndex)
		if err != nil {
			return err
		}
		err = getHelper(array[num], holder, nextPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func (j Json) Get(holder any, path string) error {
	err := getHelper(j.value, holder, path)
	if err != nil {
		return err
	}
	return nil
}

func (s stateMap) init() stateMap {
	object := &token{start: '{', end: '}', depth: 0, sep: ','}
	array := &token{start: '[', end: ']', depth: 0, sep: ','}
	value := &token{start: ':', end: ',', depth: -1}

	return stateMap{
		'{':  object,
		'}':  object,
		'[':  array,
		']':  array,
		'"':  &token{start: '"', end: '"', depth: -1},
		':':  value,
		',':  value,
		'\\': &token{start: '\\', end: '"', depth: -1},
	}
}

func Decode(raw string) (Gosonified, error) {

	var err error
	var result Json
	if raw[0] == '[' {
		result.value, err = parseArray(raw[1 : len(raw)-1])
		if err != nil {
			return nil, err
		}
	} else {
		result.value, err = parseObject(raw)
		if err != nil {
			fmt.Println("err parseObject in Decode")
			return nil, err
		}
	}
	return result, nil
}

func decoderHelper(state stateMap, stack *[]rune, depth *int, char rune, t *token, isLastChar bool, index int) error {
	if len(*stack) > 1 && state[(*stack)[len(*stack)-1]].end == ',' && state[(*stack)[len(*stack)-1]].end == '}' {
		*stack = (*stack)[:len(*stack)-1]
	}
	skip := false

	if len(*stack) > 0 && char == state[(*stack)[len(*stack)-1]].end {
		skip = true
	}

	if char == t.start && !skip {
		isUrl := len(*stack) > 1 && state[(*stack)[len(*stack)-1]].end == '"' && state[(*stack)[len(*stack)-2]].start == ':' && char == ':'
		if !isUrl {
			*stack = append(*stack, char)
			if t.depth != -1 {
				*depth++
				t.depth++
			}
		}
	} else if len(*stack) > 0 && state[(*stack)[len(*stack)-1]].sep == char {
	} else if char == t.end {
		if len(*stack) > 0 {
			if state[(*stack)[len(*stack)-1]].start != t.start && state[(*stack)[len(*stack)-1]].start != '\\' {
				if state[(*stack)[len(*stack)-1]].end == ',' && isLastChar {
					*stack = (*stack)[:len(*stack)-1]

				} else if state[(*stack)[len(*stack)-1]].end == ',' && state[(*stack)[len(*stack)-2]].end == '}' {
					*stack = (*stack)[:len(*stack)-1]

				} else {
					return errors.New("expected2: " + string(state[(*stack)[len(*stack)-1]].end) + " found: " + string(char) + " index: " + strconv.Itoa(index))
				}
			}
			*stack = (*stack)[:len(*stack)-1]

		} else {
			return errors.New("expected: EOF" + " found: " + string(char))
		}
		if t.depth != -1 {
			*depth--
			t.depth--
		}
	}
	return nil
}

func createProperty(propName string, jMap *Object) error {
	if len(propName) > 0 {
		(*jMap)[propName[:len(propName)-1]] = nil
	}
	return nil
}

func appendValue(propName string, value string, jMap any) error {
	value = strings.TrimSpace(value)
	if len(value) == 0 {
		return nil
	}
	var result any
	var err error

	if value[0] == '[' && (value[len(value)-1] == ']' || (value[len(value)-2] == ']' && value[len(value)-1] == ',')) {
		result, err = parseArray(value[1 : len(value)-1])
		if err != nil {
			fmt.Println("err parseArray in appendValue")
			return err
		}
	} else if value[0] == '"' && (value[len(value)-1] == '"' || ((value[len(value)-2] == '"' || value[len(value)-2] == '}') && value[len(value)-1] == ',')) {
		value = value[1 : len(value)-2]
		result = value
	} else if value[0] == '{' && (value[len(value)-1] == '}' || (value[len(value)-2] == '"' && value[len(value)-1] == ',')) {
		object, err := parseObject(value)
		if err != nil {
			fmt.Println("err parseObject in appendValue")
			return err
		}
		result = object
	} else {
		value = value[:len(value)-1]
		if value == "true" || value == "false" || value == "null" {
			result = value
		} else {
			num, err := strconv.Atoi(value)
			if err != nil {
				return errors.New("invalid value type: " + value)
			}
			result = num
		}
	}
	switch v := (jMap).(type) {
	case *Object:
		if len(propName) > 0 {
			(*v)[propName[:len(propName)-1]] = result
		}
	case *[]any:
		*v = append(*v, result)
	default:
		var Type reflect.Type
		Type = reflect.TypeOf(v)
		fmt.Println(v)
		return errors.New("invalid type" + Type.Name())
	}

	return nil
}

func parseArray(str string) ([]any, error) {
	result := []any{}
	state := stateMap{}.init()
	rawR := []rune(str)
	index := 0
	stack := []rune{}
	depth := 0
	arrayIndex := 0
	item := ""
	for index < len(rawR) {
		char := rawR[index]
		item += string(char)
		if item == "," || (index == len(rawR)-1 && char == ']') {
			item = ""
			index++
			continue
		}
		t, exist := state[char]
		if exist {
			err := decoderHelper(state, &stack, &depth, char, t, index == len(rawR)-1, index)
			if err != nil {
				fmt.Println("err decoderHelper in parseArray")
				return nil, err
			}
		}

		if len(stack) == 0 {

			err := appendValue("array", item, &result)
			if err != nil {
				fmt.Println("err append in parse array")
				return nil, err
			}
			arrayIndex++
			item = ""
		}

		index++
	}

	return result, nil
}

func parseObject(raw string) (Object, error) {
	raw = strings.TrimSpace(raw)
	rawR := []rune(raw[1 : len(raw)-1])
	state := stateMap{}.init()
	stack := []rune{}
	index := 0
	depth := 0
	result := Object{}
	property := ""
	var value string
	inProp := false
	inValue := false
	prevProp := ""

	for index < len(rawR) {
		char := rawR[index]
		if inProp && !inValue {
			property += string(char)
		} else if inValue {
			value += string(char)
		}
		if char == '{' {
		} else if char == '"' {
			if !inProp {
				inProp = true
			} else {
				err := createProperty(property, &result)
				if err != nil {
					fmt.Println("err createPropery in parseObject")
					return nil, err
				}
				if !inValue {
					prevProp = property
					property = ""
				}
				inProp = false
			}
		} else if char == ':' {
			inValue = true
		}
		if index > 0 && char == ',' && char == rawR[index-1] {
			return nil, errors.New("expected1: " + string(state[(stack)[len(stack)-1]].end) + " found: " + string(char))
		}
		t, exist := state[char]
		if exist {
			err := decoderHelper(state, &stack, &depth, char, t, index == len(rawR)-1, index)
			if err != nil {
				fmt.Println("err decoderHelper in parseObject")
				return nil, err
			}
		}
		if len(stack) > 0 && index == len(rawR)-1 && stack[len(stack)-1] == ':' {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 && len(value) > 0 {
			err := appendValue(prevProp, value, &result)
			if err != nil {
				fmt.Println("err append in parse object")
				return nil, err
			}
			value = ""
			inValue = false
		}
		index++
	}
	if depth != 0 {
		return nil, fmt.Errorf("mismatched brackets: depth is %d at the end of parsing, expected: %q", depth, state[stack[len(stack)-1]].end)
	}
	return result, nil
}
