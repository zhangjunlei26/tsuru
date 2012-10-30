// Copyright 2012 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"github.com/globocom/tsuru/cmd"
	. "launchpad.net/gocheck"
	"net/http"
	"strings"
)

func (s *S) TestAppLog(c *C) {
	*appName = "appName"
	var stdout, stderr bytes.Buffer
	result := `[{"Date":"2012-06-20T11:17:22.75-03:00","Message":"creating app lost"},{"Date":"2012-06-20T11:17:22.753-03:00","Message":"app lost successfully created"}]`
	expected := `2012-06-20 11:17:22.75 -0300 BRT - creating app lost
2012-06-20 11:17:22.753 -0300 BRT - app lost successfully created
`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: result, status: http.StatusOK}}, nil, "", "")
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	got := stdout.String()
	got = strings.Replace(got, "-0300 -0300", "-0300 BRT", -1)
	c.Assert(got, Equals, expected)
}

func (s *S) TestAppLogWithoutTheFlag(c *C) {
	var stdout, stderr bytes.Buffer
	result := `[{"Date":"2012-06-20T11:17:22.75-03:00","Message":"creating app lost"},{"Date":"2012-06-20T11:17:22.753-03:00","Message":"app lost successfully created"}]`
	expected := `2012-06-20 11:17:22.75 -0300 BRT - creating app lost
2012-06-20 11:17:22.753 -0300 BRT - app lost successfully created
`
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &FakeGuesser{name: "hitthelights"}
	command := AppLog{GuessingCommand{g: fake}}
	trans := &conditionalTransport{
		transport{
			msg:    result,
			status: http.StatusOK,
		},
		func(req *http.Request) bool {
			return req.URL.Path == "/apps/hitthelights/log" && req.Method == "GET"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, "", "")
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	got := stdout.String()
	got = strings.Replace(got, "-0300 -0300", "-0300 BRT", -1)
	c.Assert(got, Equals, expected)
}

func (s *S) TestAppLogShouldReturnNilIfHasNoContent(c *C) {
	*appName = "appName"
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := AppLog{}
	client := cmd.NewClient(&http.Client{Transport: &transport{msg: "", status: http.StatusNoContent}}, nil, "", "")
	err := command.Run(&context, client)
	c.Assert(err, IsNil)
	c.Assert(stdout.String(), Equals, "")
}

func (s *S) TestAppLogInfo(c *C) {
	expected := &cmd.Info{
		Name:  "log",
		Usage: "log [--app appname]",
		Desc: `show logs for an app.

If you don't provide the app name, tsuru will try to guess it.`,
		MinArgs: 0,
	}
	c.Assert((&AppLog{}).Info(), DeepEquals, expected)
}