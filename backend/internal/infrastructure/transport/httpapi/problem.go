package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"ant/internal/application/platform"
	"ant/internal/contracts/errcodes"
)

// Problem — тело ответа об ошибке: RFC 9457 application/problem+json с
// расширениями ant (contracts/problem.schema.json, FR-28). Код — из
// contracts/errors.yaml; type = urn:ant:problem:‹код›.
type Problem struct {
	Type           string            `json:"type" doc:"URI типа проблемы: urn:ant:problem:‹код›."`
	Title          string            `json:"title" doc:"Краткий заголовок по-русски (из каталога)."`
	Status         int               `json:"status" minimum:"200" maximum:"599" doc:"HTTP-статус."`
	Detail         string            `json:"detail,omitempty" doc:"Пояснение по-русски для этого случая."`
	Instance       string            `json:"instance,omitempty" doc:"URI конкретного случая (путь запроса или id записи карантина)."`
	Code           string            `json:"code" doc:"Код из contracts/errors.yaml."`
	Params         map[string]string `json:"params,omitempty" doc:"Параметры шаблона текста интерфейса."`
	Violations     []Violation       `json:"violations,omitempty" doc:"Нарушения по полям или элементам."`
	QuarantineID   string            `json:"quarantine_id,omitempty" doc:"Запись карантина (FR-30)."`
	CARef          string            `json:"ca_ref,omitempty" doc:"Критическое действие, записанное из-за отказа."`
	BasisSeq       int64             `json:"basis_seq,omitempty" doc:"Для journal.stale_*: на каком seq проверено."`
	AllowedActions []string          `json:"allowed_actions,omitempty" doc:"Что пользователь может сделать вместо: например, «Запросить решение» (FR-146)."`
}

// Violation — нарушение по полю или элементу.
type Violation struct {
	Code      string `json:"code" doc:"Код нарушения."`
	Field     string `json:"field,omitempty" doc:"JSON Pointer поля."`
	ElementID string `json:"element_id,omitempty" doc:"Идентификатор элемента BPMN."`
	Message   string `json:"message,omitempty" doc:"Пояснение."`
}

// Error реализует error.
func (p *Problem) Error() string { return p.Code + ": " + p.Detail }

// GetStatus реализует huma.StatusError.
func (p *Problem) GetStatus() int { return p.Status }

// ContentType — problem+json (RFC 9457).
func (p *Problem) ContentType(ct string) string {
	if ct == "application/json" {
		return "application/problem+json"
	}
	return ct
}

// newProblem — проблема с кодом из каталога.
func newProblem(code errcodes.Code, detail string) *Problem {
	info, ok := errcodes.Lookup(code)
	if !ok {
		info = errcodes.Info{Code: code, Status: http.StatusInternalServerError, Title: string(code)}
	}
	return &Problem{
		Type:   errcodes.ProblemTypePrefix + string(code),
		Title:  info.Title,
		Status: info.Status,
		Detail: detail,
		Code:   string(code),
	}
}

// problemFrom переводит ошибку порта или доменный отказ в Problem (FR-28).
func problemFrom(err error, operationID string) error {
	var p *Problem
	if errors.As(err, &p) {
		return p
	}
	var se huma.StatusError
	if errors.As(err, &se) {
		return err
	}
	if e, ok := platform.AsError(err); ok {
		pr := newProblem(e.Code, e.Detail)
		if pr.Detail == "" {
			if info, ok := errcodes.Lookup(e.Code); ok {
				pr.Detail = fill(info.Detail, e.Params)
			}
		}
		pr.Params = e.Params
		pr.AllowedActions = e.AllowedActions
		pr.BasisSeq = e.BasisSeq
		pr.CARef = e.CARef
		return pr
	}
	pr := newProblem(errcodes.ApiInternalError, "Внутренняя ошибка при выполнении "+operationID)
	return pr
}

func fill(tpl string, params map[string]string) string {
	for k, v := range params {
		tpl = strings.ReplaceAll(tpl, "{"+k+"}", v)
	}
	return tpl
}

func platformFail(code string) error { return platform.Fail(errcodes.Code(code)) }

func platformValidation(field, reason string) error {
	return platform.Fail(errcodes.ApiValidationFailed, "field", field, "reason", reason)
}

// statusCode — код по умолчанию для ошибок самого Huma (маршрутизация, разбор, проверка схемы).
func statusCode(status int) errcodes.Code {
	switch status {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return errcodes.ApiValidationFailed
	case http.StatusUnauthorized:
		return errcodes.AccessUnauthenticated
	case http.StatusForbidden:
		return errcodes.AccessForbidden
	case http.StatusNotFound:
		return errcodes.ApiNotFound
	case http.StatusTooManyRequests:
		return errcodes.ApiRateLimited
	case http.StatusNotImplemented:
		return errcodes.ApiNotImplemented
	default:
		return errcodes.ApiInternalError
	}
}

// init подменяет конструктор ошибок Huma: все ошибки — Problem с кодом (FR-28);
// им же Huma описывает ответы об ошибках в спецификации.
func init() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		p := newProblem(statusCode(status), msg)
		if status != 0 {
			p.Status = status
		}
		for _, e := range errs {
			if e == nil {
				continue
			}
			v := Violation{Code: string(errcodes.ApiValidationFailed), Message: e.Error()}
			var d huma.ErrorDetailer
			if errors.As(e, &d) {
				det := d.ErrorDetail()
				v.Message = det.Message
				v.Field = "/" + strings.ReplaceAll(strings.ReplaceAll(det.Location, ".", "/"), "[", "/")
				v.Field = strings.ReplaceAll(v.Field, "]", "")
			}
			p.Violations = append(p.Violations, v)
		}
		return p
	}
}
