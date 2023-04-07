package sqlparams

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	cQuestionParams = iota + 1
	cNumericParams
	cNamedParams

	tmFmtWithMS = "2006-01-02 15:04:05.999"
	tmFmtZero   = "0000-00-00 00:00:00"
	nullStr     = "NULL"
)

var (
	numericPlaceholder = regexp.MustCompile(`\$(\d+)`)             // postgres
	namedPlaceholder   = regexp.MustCompile(`([^:])(:[a-z0-9_]+)`) // for named params
	convertableTypes   = []reflect.Type{reflect.TypeOf(time.Time{}), reflect.TypeOf(false), reflect.TypeOf([]byte{})}
	escaper            = `'`
)

type CustomContraint interface {
	int | int8 | int16 | int32 | int64 | uint | uint16 | uint32 | uint64 | string
}

func Inline(sql string, avars ...any) string {
	avarsLen := len(avars)
	if avarsLen == 0 {
		return sql
	}

	var (
		convertParams func(any, any)
		vars          = make(map[any]string, len(avars))
		placeholder   int
	)

	switch {
	case strings.Contains(sql, `?`):
		placeholder = cQuestionParams
	case numericPlaceholder.MatchString(sql):
		placeholder = cNumericParams
	case namedPlaceholder.MatchString(sql):
		placeholder = cNamedParams
	default:
		return `placeholder is undefined: ` + sql
	}

	convertParams = func(v any, idx any) {
		switch v := v.(type) {
		case bool:
			vars[idx] = strconv.FormatBool(v)
		case time.Time:
			if v.IsZero() {
				vars[idx] = escaper + tmFmtZero + escaper
			} else {
				vars[idx] = escaper + v.Format(tmFmtWithMS) + escaper
			}
		case *time.Time:
			if v != nil {
				if v.IsZero() {
					vars[idx] = escaper + tmFmtZero + escaper
				} else {
					vars[idx] = escaper + v.Format(tmFmtWithMS) + escaper
				}
			} else {
				vars[idx] = nullStr
			}
		case driver.Valuer:
			reflectValue := reflect.ValueOf(v)
			if v != nil && reflectValue.IsValid() && ((reflectValue.Kind() == reflect.Ptr && !reflectValue.IsNil()) || reflectValue.Kind() != reflect.Ptr) {
				r, _ := v.Value()
				convertParams(r, idx)
			} else {
				vars[idx] = nullStr
			}
		case fmt.Stringer:
			reflectValue := reflect.ValueOf(v)
			if v != nil && reflectValue.IsValid() && ((reflectValue.Kind() == reflect.Ptr && !reflectValue.IsNil()) || reflectValue.Kind() != reflect.Ptr) {
				vars[idx] = escaper + strings.Replace(fmt.Sprintf("%v", v), escaper, "\\"+escaper, -1) + escaper
			} else {
				vars[idx] = nullStr
			}
		case []byte:
			if isPrintable(v) {
				vars[idx] = escaper + strings.Replace(string(v), escaper, "\\"+escaper, -1) + escaper
			} else {
				vars[idx] = escaper + "<binary>" + escaper
			}
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			vars[idx] = toString(v)
		case float64, float32:
			vars[idx] = fmt.Sprintf("%.6f", v)
		case string:
			vars[idx] = escaper + strings.Replace(v, escaper, "\\"+escaper, -1) + escaper
		case []string:
			vars[idx] = sliceToArray(v)
		case []int:
			vars[idx] = sliceToArray(v)
		case []int8:
			vars[idx] = sliceToArray(v)
		case []int16:
			vars[idx] = sliceToArray(v)
		case []int32:
			vars[idx] = sliceToArray(v)
		case []int64:
			vars[idx] = sliceToArray(v)
		case []uint:
			vars[idx] = sliceToArray(v)
		case []uint16:
			vars[idx] = sliceToArray(v)
		case []uint32:
			vars[idx] = sliceToArray(v)
		case []uint64:
			vars[idx] = sliceToArray(v)
		default:
			rv := reflect.ValueOf(v)
			if v == nil || !rv.IsValid() || rv.Kind() == reflect.Ptr && rv.IsNil() {
				vars[idx] = nullStr
			} else if valuer, ok := v.(driver.Valuer); ok {
				v, _ = valuer.Value()
				convertParams(v, idx)
			} else if rv.Kind() == reflect.Ptr && !rv.IsZero() {
				convertParams(reflect.Indirect(rv).Interface(), idx)
			} else {
				for _, t := range convertableTypes {
					if rv.Type().ConvertibleTo(t) {
						convertParams(rv.Convert(t).Interface(), idx)
						return
					}
				}
				vars[idx] = escaper + strings.Replace(fmt.Sprint(v), escaper, "\\"+escaper, -1) + escaper
			}
		}
	}

	if avarsLen == 1 {
		//
		// reflect parameters
		//
		var ptr bool
		var timeType = reflect.TypeOf(time.Time{})
		t := reflect.TypeOf(avars[0])
		val := reflect.ValueOf(avars[0])
		if t.Kind() == reflect.Ptr {
			val = val.Elem()
			t = t.Elem()
			ptr = true
		}
		switch {
		case t.Kind() == reflect.Struct:
			if t == timeType || (t.Kind() == reflect.Ptr && t.Elem() == timeType) { // time.Time
				convertParams(avars[0], 0)
			} else if _, ok := avars[0].(driver.Valuer); ok {
				convertParams(avars[0], 0)
			} else if _, ok := avars[0].(fmt.Stringer); ok {
				convertParams(avars[0], 0)
			} else { // a struct as parameter
				var (
					key string
					v   any
					num int // index for numeric params
				)
				for i := 0; i < val.NumField(); i++ {
					if !val.Field(i).CanInterface() { // ignore if unexported
						continue
					}
					if t.Field(i).Type.Kind() == reflect.Ptr {
						if val.Field(i).IsNil() {
							v = nil
						} else {
							v = val.Field(i).Elem().Interface()
						}
						if len(t.Field(i).Tag.Get("db")) > 0 {
							key = t.Field(i).Tag.Get("db")
						} else {
							key = strings.ToLower(t.Field(i).Name)
						}

					} else {
						valueField := val.Field(i)
						typeField := val.Type().Field(i)
						tag := typeField.Tag
						if len(tag.Get("db")) > 0 {
							key = tag.Get("db")
						} else {
							key = strings.ToLower(typeField.Name)
						}
						v = valueField.Interface()
					}
					if placeholder == cNamedParams {
						convertParams(v, key) // index - field name for named params
					} else {
						convertParams(v, num) // index - incremented digit
						num++
					}
				}
			}

		case t.Kind() == reflect.Map && t.Key().Kind() == reflect.String:
			var p any
			if ptr {
				p = *avars[0].(*map[string]any) // for a special case...
			} else {
				p = avars[0].(map[string]any)
			}
			for key, v := range p.(map[string]any) {
				convertParams(v, key)
			}
		default:
			convertParams(avars[0], 0)
		}

	} else {
		// params list
		for idx, v := range avars {
			convertParams(v, idx)
		}
	}

	switch placeholder {
	case cNumericParams:
		sql = numericPlaceholder.ReplaceAllString(sql, "$$$1$$")
		for idx, v := range vars {
			if _, ok := idx.(int); !ok {
				break
			}
			sql = strings.Replace(sql, "$"+strconv.Itoa(idx.(int)+1)+"$", v, -1)
		}
	case cNamedParams:
		sql = namedPlaceholder.ReplaceAllString(sql, "$1$$$2$$")
		for idx, v := range vars {
			if _, ok := idx.(string); !ok {
				break
			}
			sql = strings.Replace(sql, "$:"+idx.(string)+"$", v, -1)
		}
	default: // cQuestionParams
		var idx int
		var newSQL strings.Builder
		for _, v := range []byte(sql) {
			if v == '?' {
				if len(vars) > idx {
					newSQL.WriteString(vars[idx])
					idx++
					continue
				}
			}
			newSQL.WriteByte(v)
		}
		sql = newSQL.String()
	}

	return sql
}

func isPrintable(s []byte) bool {
	for _, r := range s {
		if !unicode.IsPrint(rune(r)) {
			return false
		}
	}
	return true
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, escaper) {
			if strings.Contains(v, `\`+escaper) {
				return escaper + v + escaper
			}
			return escaper + strings.ReplaceAll(v, escaper, `\`+escaper) + escaper
		}
		return escaper + v + escaper
	case int:
		return strconv.FormatInt(int64(v), 10)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	}
	return ""
}

func sliceToArray[T CustomContraint](v []T) string {
	var str = make([]string, 0, len(v))
	for i := range v {
		str = append(str, toString(v[i]))
	}
	return `ARRAY[` + strings.Join(str, `, `) + `]`

}
