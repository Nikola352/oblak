package service

import "github.com/google/uuid"

type BuildMessage struct {
	FunctionId     uuid.UUID `json:"function_id"`
	CodeObjectName string    `json:"code_object_name"`
}

type ExecuteMessage struct {
	InvocationId           uuid.UUID `json:"invocation_id"`
	CodeObjectName         string    `json:"code_object_name"`
	DependenciesObjectName string    `json:"dependencies_object_name"`
}
