package validator

import (
	"regexp"

	"github.com/dongwlin/legero-backend/internal/pkg/errs"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/gofiber/fiber/v2"
)

var (
	phoneNumRegexp = regexp.MustCompile(`^1\d{10}$`)
)

var Validate = New()

var (
	uni   = ut.New(en.New())
	trans ut.Translator
)

func init() {
	trans, _ = uni.GetTranslator("en")
	en_translations.RegisterDefaultTranslations(Validate, trans)

	Validate.RegisterTranslation("phone_num", trans, func(ut ut.Translator) error {
		return ut.Add("phone_num", "{0} must be a valid phone number", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("phone_num", fe.Field())
		return t
	})
}

func New() *validator.Validate {

	validate := validator.New()

	validate.RegisterValidation("phone_num", phoneNum)

	return validate
}

func phoneNum(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	return phoneNumRegexp.MatchString(val)
}

type ValidationError struct {
	Field     string `json:"field"`
	Violation string `json:"violation"`
	Message   string `json:"message"`
}

func convertValidationErrors(ves validator.ValidationErrors) []*ValidationError {

	errors := make([]*ValidationError, 0, len(ves))

	for _, fe := range ves {

		errors = append(errors, &ValidationError{
			Field:     fe.Field(),
			Violation: fe.Tag(),
			Message:   fe.Translate(trans),
		})
	}

	return errors
}

func ValidateBody(c *fiber.Ctx, dest any) error {

	if err := c.BodyParser(dest); err != nil {
		return &errs.Error{
			StatusCode: fiber.StatusBadRequest,
			Message:    "invalid params",
		}
	}

	if err := Validate.Struct(dest); err != nil {

		ves, ok := err.(validator.ValidationErrors)
		if !ok {
			panic(err)
		}

		validationErrors := convertValidationErrors(ves)

		return &errs.Error{
			StatusCode: fiber.StatusBadRequest,
			Message:    "invalid params",
			Data: fiber.Map{
				"violations": validationErrors,
			},
		}
	}

	return nil
}
