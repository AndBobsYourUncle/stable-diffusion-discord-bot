package repositories

import "fmt"

type NotFoundError struct {
	entityName string
}

func NewNotFoundError(entityName string) *NotFoundError {
	return &NotFoundError{entityName: entityName}
}

func (m *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", m.entityName)
}

func (e *NotFoundError) Is(err error) bool {
	_, ok := err.(*NotFoundError)
	return ok
}
