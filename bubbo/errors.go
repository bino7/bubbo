package bubbo

import "errors"

var (
	ErrorForbidden=errors.New("forbidden")
	ErrorUserExisted=errors.New("user existed")
	ErrorUserNotFound =errors.New("user not found")
	ErrorEmptyField=errors.New("empty field")
	ErrorCheckUserInfo=errors.New("check userInfo error")
	ErrUpdateUserInfoFailed=errors.New("update userInfo failed")
	ErrBadMsg=errors.New("bad msg")
	ErrRemoteNotFound=errors.New("remote not found")
	ErrorMediaNotFound=errors.New("user not found")
)

