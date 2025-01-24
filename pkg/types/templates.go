package types

import "github.com/google/uuid"

var UUID = func() string { return uuid.New().String() }