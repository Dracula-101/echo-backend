package validation

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type FieldError struct {
	Field   string `json:"field"`
	Value   any    `json:"value,omitempty"`
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

type ValidationErrors []FieldError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return ""
	}
	var sb strings.Builder
	for i, err := range v {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(fmt.Sprintf("%s: %s", err.Field, err.Message))
	}
	return sb.String()
}

type RuleFunc func(value interface{}, params ...string) error

type Validator struct {
	rules map[string]RuleFunc
}

func New() *Validator {
	v := &Validator{
		rules: make(map[string]RuleFunc),
	}
	v.registerDefaultRules()
	return v
}

func (v *Validator) RegisterRule(name string, rule RuleFunc) {
	v.rules[name] = rule
}

func (v *Validator) Validate(data interface{}, rules map[string]string) ValidationErrors {
	var errors ValidationErrors
	val := reflect.ValueOf(data)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return ValidationErrors{{
			Field:   "root",
			Rule:    "struct",
			Message: "data must be a struct",
		}}
	}

	for fieldName, ruleStr := range rules {
		field := val.FieldByName(fieldName)
		if !field.IsValid() {
			errors = append(errors, FieldError{
				Field:   fieldName,
				Rule:    "exists",
				Message: "field not found",
			})
			continue
		}

		fieldRules := parseRules(ruleStr)
		for _, rule := range fieldRules {
			if ruleFunc, exists := v.rules[rule.name]; exists {
				if err := ruleFunc(field.Interface(), rule.params...); err != nil {
					errors = append(errors, FieldError{
						Field:   fieldName,
						Value:   field.Interface(),
						Rule:    rule.name,
						Message: err.Error(),
					})
					break
				}
			}
		}
	}

	return errors
}

type rule struct {
	name   string
	params []string
}

func parseRules(ruleStr string) []rule {
	var rules []rule
	parts := strings.Split(ruleStr, "|")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		if idx := strings.Index(part, ":"); idx != -1 {
			name := part[:idx]
			params := strings.Split(part[idx+1:], ",")
			for i := range params {
				params[i] = strings.TrimSpace(params[i])
			}
			rules = append(rules, rule{name: name, params: params})
		} else {
			rules = append(rules, rule{name: part})
		}
	}

	return rules
}

func (v *Validator) registerDefaultRules() {
	v.RegisterRule("required", ruleRequired)
	v.RegisterRule("email", ruleEmail)
	v.RegisterRule("min", ruleMin)
	v.RegisterRule("max", ruleMax)
	v.RegisterRule("minlen", ruleMinLen)
	v.RegisterRule("maxlen", ruleMaxLen)
	v.RegisterRule("len", ruleLen)
	v.RegisterRule("regex", ruleRegex)
	v.RegisterRule("in", ruleIn)
	v.RegisterRule("notin", ruleNotIn)
	v.RegisterRule("numeric", ruleNumeric)
	v.RegisterRule("alpha", ruleAlpha)
	v.RegisterRule("alphanumeric", ruleAlphanumeric)
	v.RegisterRule("url", ruleURL)
	v.RegisterRule("uuid", ruleUUID)
	v.RegisterRule("phone", rulePhone)
	v.RegisterRule("ip", ruleIP)
	v.RegisterRule("ipv4", ruleIPv4)
	v.RegisterRule("ipv6", ruleIPv6)
	v.RegisterRule("date", ruleDate)
	v.RegisterRule("datetime", ruleDateTime)
	v.RegisterRule("slug", ruleSlug)
	v.RegisterRule("json", ruleJSON)
}

func ruleRequired(value interface{}, params ...string) error {
	if value == nil {
		return fmt.Errorf("field is required")
	}

	val := reflect.ValueOf(value)
	switch val.Kind() {
	case reflect.String:
		if val.String() == "" {
			return fmt.Errorf("field is required")
		}
	case reflect.Slice, reflect.Map, reflect.Array:
		if val.Len() == 0 {
			return fmt.Errorf("field is required")
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val.Int() == 0 {
			return fmt.Errorf("field is required")
		}
	case reflect.Ptr:
		if val.IsNil() {
			return fmt.Errorf("field is required")
		}
	}

	return nil
}

func ruleEmail(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(str) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func ruleMin(value interface{}, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("min rule requires a parameter")
	}

	minVal, err := strconv.ParseFloat(params[0], 64)
	if err != nil {
		return fmt.Errorf("invalid min parameter")
	}

	val := reflect.ValueOf(value)
	var numVal float64

	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		numVal = float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		numVal = float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		numVal = val.Float()
	default:
		return fmt.Errorf("must be numeric")
	}

	if numVal < minVal {
		return fmt.Errorf("must be at least %s", params[0])
	}

	return nil
}

func ruleMax(value interface{}, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("max rule requires a parameter")
	}

	maxVal, err := strconv.ParseFloat(params[0], 64)
	if err != nil {
		return fmt.Errorf("invalid max parameter")
	}

	val := reflect.ValueOf(value)
	var numVal float64

	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		numVal = float64(val.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		numVal = float64(val.Uint())
	case reflect.Float32, reflect.Float64:
		numVal = val.Float()
	default:
		return fmt.Errorf("must be numeric")
	}

	if numVal > maxVal {
		return fmt.Errorf("must be at most %s", params[0])
	}

	return nil
}

func ruleMinLen(value interface{}, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("minlen rule requires a parameter")
	}

	minLen, err := strconv.Atoi(params[0])
	if err != nil {
		return fmt.Errorf("invalid minlen parameter")
	}

	val := reflect.ValueOf(value)
	var length int

	switch val.Kind() {
	case reflect.String:
		length = len(val.String())
	case reflect.Slice, reflect.Array, reflect.Map:
		length = val.Len()
	default:
		return fmt.Errorf("must be a string, slice, array, or map")
	}

	if length < minLen {
		return fmt.Errorf("must be at least %d characters/items", minLen)
	}

	return nil
}

func ruleMaxLen(value interface{}, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("maxlen rule requires a parameter")
	}

	maxLen, err := strconv.Atoi(params[0])
	if err != nil {
		return fmt.Errorf("invalid maxlen parameter")
	}

	val := reflect.ValueOf(value)
	var length int

	switch val.Kind() {
	case reflect.String:
		length = len(val.String())
	case reflect.Slice, reflect.Array, reflect.Map:
		length = val.Len()
	default:
		return fmt.Errorf("must be a string, slice, array, or map")
	}

	if length > maxLen {
		return fmt.Errorf("must be at most %d characters/items", maxLen)
	}

	return nil
}

func ruleLen(value interface{}, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("len rule requires a parameter")
	}

	expectedLen, err := strconv.Atoi(params[0])
	if err != nil {
		return fmt.Errorf("invalid len parameter")
	}

	val := reflect.ValueOf(value)
	var length int

	switch val.Kind() {
	case reflect.String:
		length = len(val.String())
	case reflect.Slice, reflect.Array, reflect.Map:
		length = val.Len()
	default:
		return fmt.Errorf("must be a string, slice, array, or map")
	}

	if length != expectedLen {
		return fmt.Errorf("must be exactly %d characters/items", expectedLen)
	}

	return nil
}

func ruleRegex(value interface{}, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("regex rule requires a parameter")
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	regex, err := regexp.Compile(params[0])
	if err != nil {
		return fmt.Errorf("invalid regex pattern")
	}

	if !regex.MatchString(str) {
		return fmt.Errorf("invalid format")
	}

	return nil
}

func ruleIn(value interface{}, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("in rule requires parameters")
	}

	str := fmt.Sprintf("%v", value)

	for _, param := range params {
		if str == param {
			return nil
		}
	}

	return fmt.Errorf("must be one of: %s", strings.Join(params, ", "))
}

func ruleNotIn(value interface{}, params ...string) error {
	if len(params) == 0 {
		return fmt.Errorf("notin rule requires parameters")
	}

	str := fmt.Sprintf("%v", value)

	for _, param := range params {
		if str == param {
			return fmt.Errorf("must not be one of: %s", strings.Join(params, ", "))
		}
	}

	return nil
}

func ruleNumeric(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	if _, err := strconv.ParseFloat(str, 64); err != nil {
		return fmt.Errorf("must be numeric")
	}

	return nil
}

func ruleAlpha(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	alphaRegex := regexp.MustCompile(`^[a-zA-Z]+$`)
	if !alphaRegex.MatchString(str) {
		return fmt.Errorf("must contain only letters")
	}

	return nil
}

func ruleAlphanumeric(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	alphanumericRegex := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !alphanumericRegex.MatchString(str) {
		return fmt.Errorf("must contain only letters and numbers")
	}

	return nil
}

func ruleURL(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	if !urlRegex.MatchString(str) {
		return fmt.Errorf("invalid URL format")
	}

	return nil
}

func ruleUUID(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !uuidRegex.MatchString(str) {
		return fmt.Errorf("invalid UUID format")
	}

	return nil
}

func rulePhone(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	if !phoneRegex.MatchString(str) {
		return fmt.Errorf("invalid phone number format")
	}

	return nil
}

func ruleIP(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)

	if !ipv4Regex.MatchString(str) && !ipv6Regex.MatchString(str) {
		return fmt.Errorf("invalid IP address format")
	}

	return nil
}

func ruleIPv4(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	ipv4Regex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	if !ipv4Regex.MatchString(str) {
		return fmt.Errorf("invalid IPv4 address format")
	}

	return nil
}

func ruleIPv6(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	ipv6Regex := regexp.MustCompile(`^([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}$`)
	if !ipv6Regex.MatchString(str) {
		return fmt.Errorf("invalid IPv6 address format")
	}

	return nil
}

func ruleDate(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	dateRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	if !dateRegex.MatchString(str) {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}

	return nil
}

func ruleDateTime(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	dateTimeRegex := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)
	if !dateTimeRegex.MatchString(str) {
		return fmt.Errorf("invalid datetime format, expected ISO 8601")
	}

	return nil
}

func ruleSlug(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	slugRegex := regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	if !slugRegex.MatchString(str) {
		return fmt.Errorf("invalid slug format")
	}

	return nil
}

func ruleJSON(value interface{}, params ...string) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	var js interface{}
	if err := json.Unmarshal([]byte(str), &js); err != nil {
		return fmt.Errorf("invalid JSON format")
	}

	return nil
}
