package bubbo

import (
	"net/http"
)

type BubboError struct {
	code int
	msg  string
}

func (e *BubboError)Error() string {
	return e.msg
}

var (
	ErrorForbidden = &BubboError{http.StatusForbidden, "forbidden"}
	ErrorUserExisted = &BubboError{http.StatusConflict, "user existed"}
	ErrorUserNotFound = &BubboError{http.StatusNotFound, "user not found"}
	ErrorEmptyField = &BubboError{http.StatusNoContent, "empty field"}
	ErrBadMsg = &BubboError{http.StatusNoContent, "bad msg"}
	ErrRemoteNotFound = &BubboError{http.StatusNotFound, "remote not found"}
	ErrorMediaNotFound = &BubboError{http.StatusNotFound, "user not found"}
	ErrorNotModified = &BubboError{http.StatusNotModified, "not updated"}
)

