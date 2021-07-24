// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build generate

package registry

//go:generate go run github.com/yzy613/ddns-watchdog/golang.org/x/sys/windows/mkwinsyscall -output zsyscall_windows.go syscall.go
