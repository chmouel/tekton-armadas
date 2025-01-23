package templates

import "github.com/google/uuid"

var UUID = func() string { return uuid.New().String() }
