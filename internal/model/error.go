package model

import "errors"

var ErrURLAlreadyExists = errors.New("already exists")
var ErrURLDeleted = errors.New("URL has been deleted")
