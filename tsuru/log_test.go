// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/tsuru/tsuru/cmd"
	"github.com/tsuru/tsuru/cmd/cmdtest"
	"gopkg.in/check.v1"
)

func (s *S) TestFormatterUsesCurrentTimeZone(c *check.C) {
	t := time.Now()
	logs := []log{
		{Date: t, Message: "Something happened", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "Something happened again", Source: "tsuru"},
	}
	data, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	var writer bytes.Buffer
	old := time.Local
	time.Local = time.UTC
	defer func() {
		time.Local = old
	}()
	formatter := logFormatter{}
	err = formatter.Format(&writer, data)
	c.Assert(err, check.IsNil)
	tfmt := "2006-01-02 15:04:05 -0700"
	t = t.In(time.UTC)
	expected := cmd.Colorfy(t.Format(tfmt)+" [tsuru]:", "blue", "", "") + " Something happened\n"
	expected = expected + cmd.Colorfy(t.Add(2*time.Hour).Format(tfmt)+" [tsuru]:", "blue", "", "") + " Something happened again\n"
	c.Assert(writer.String(), check.Equals, expected)
}

func (s *S) TestAppLog(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "app", Unit: "abcdef"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = t.In(time.Local)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := cmd.Colorfy(t.Format(tfmt)+" [tsuru]:", "blue", "", "") + " creating app lost\n"
	expected = expected + cmd.Colorfy(t.Add(2*time.Hour).Format(tfmt)+" [app][abcdef]:", "blue", "", "") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := appLog{}
	transport := cmdtest.Transport{
		Message: string(result),
		Status:  http.StatusOK,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command.Flags().Parse(true, []string{"--app", "appName"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithUnparsableData(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = t.In(time.Local)
	tfmt := "2006-01-02 15:04:05 -0700"

	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := appLog{}
	transport := cmdtest.Transport{
		Message: string(result) + "\nunparseable data",
		Status:  http.StatusOK,
	}
	client := cmd.NewClient(&http.Client{Transport: &transport}, nil, manager)
	command.Flags().Parse(true, []string{"--app", "appName"})
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	expected := cmd.Colorfy(t.Format(tfmt)+" [tsuru]:", "blue", "", "") + " creating app lost\n"
	expected += "Error: unparseable data"
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithoutTheFlag(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "app"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = t.In(time.Local)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := cmd.Colorfy(t.Format(tfmt)+" [tsuru]:", "blue", "", "") + " creating app lost\n"
	expected = expected + cmd.Colorfy(t.Add(2*time.Hour).Format(tfmt)+" [app]:", "blue", "", "") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &cmdtest.FakeGuesser{Name: "hitthelights"}
	command := appLog{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, nil)
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Path == "/apps/hitthelights/log" && req.Method == "GET" &&
				req.URL.Query().Get("lines") == "10"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogShouldReturnNilIfHasNoContent(c *check.C) {
	var stdout, stderr bytes.Buffer
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	command := appLog{}
	client := cmd.NewClient(&http.Client{Transport: &cmdtest.Transport{Message: "", Status: http.StatusNoContent}}, nil, manager)
	command.Flags().Parse(true, []string{"--app", "appName"})
	err := command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, "")
}

func (s *S) TestAppLogInfo(c *check.C) {
	c.Assert((&appLog{}).Info(), check.NotNil)
}

func (s *S) TestAppLogBySource(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "tsuru"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = t.In(time.Local)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := cmd.Colorfy(t.Format(tfmt)+" [tsuru]:", "blue", "", "") + " creating app lost\n"
	expected = expected + cmd.Colorfy(t.Add(2*time.Hour).Format(tfmt)+" [tsuru]:", "blue", "", "") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &cmdtest.FakeGuesser{Name: "hitthelights"}
	command := appLog{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, []string{"--source", "mysource"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("source") == "mysource"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogByUnit(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru", Unit: "api"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "tsuru", Unit: "api"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = t.In(time.Local)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := cmd.Colorfy(t.Format(tfmt)+" [tsuru][api]:", "blue", "", "") + " creating app lost\n"
	expected = expected + cmd.Colorfy(t.Add(2*time.Hour).Format(tfmt)+" [tsuru][api]:", "blue", "", "") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &cmdtest.FakeGuesser{Name: "hitthelights"}
	command := appLog{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, []string{"--unit", "api"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("unit") == "api"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithLines(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "tsuru"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = t.In(time.Local)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := cmd.Colorfy(t.Format(tfmt)+" [tsuru]:", "blue", "", "") + " creating app lost\n"
	expected = expected + cmd.Colorfy(t.Add(2*time.Hour).Format(tfmt)+" [tsuru]:", "blue", "", "") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &cmdtest.FakeGuesser{Name: "hitthelights"}
	command := appLog{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, []string{"--lines", "12"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("lines") == "12"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogWithFollow(c *check.C) {
	var stdout, stderr bytes.Buffer
	t := time.Now()
	logs := []log{
		{Date: t, Message: "creating app lost", Source: "tsuru"},
		{Date: t.Add(2 * time.Hour), Message: "app lost successfully created", Source: "tsuru"},
	}
	result, err := json.Marshal(logs)
	c.Assert(err, check.IsNil)
	t = t.In(time.Local)
	tfmt := "2006-01-02 15:04:05 -0700"
	expected := cmd.Colorfy(t.Format(tfmt)+" [tsuru]:", "blue", "", "") + " creating app lost\n"
	expected = expected + cmd.Colorfy(t.Add(2*time.Hour).Format(tfmt)+" [tsuru]:", "blue", "", "") + " app lost successfully created\n"
	context := cmd.Context{
		Stdout: &stdout,
		Stderr: &stderr,
	}
	fake := &cmdtest.FakeGuesser{Name: "hitthelights"}
	command := appLog{GuessingCommand: cmd.GuessingCommand{G: fake}}
	command.Flags().Parse(true, []string{"--lines", "12", "-f"})
	trans := &cmdtest.ConditionalTransport{
		Transport: cmdtest.Transport{Message: string(result), Status: http.StatusOK},
		CondFunc: func(req *http.Request) bool {
			return req.URL.Query().Get("lines") == "12" && req.URL.Query().Get("follow") == "1"
		},
	}
	client := cmd.NewClient(&http.Client{Transport: trans}, nil, manager)
	err = command.Run(&context, client)
	c.Assert(err, check.IsNil)
	c.Assert(stdout.String(), check.Equals, expected)
}

func (s *S) TestAppLogFlagSet(c *check.C) {
	command := appLog{}
	flagset := command.Flags()
	flagset.Parse(true, []string{"--source", "tsuru", "--unit", "abcdef", "--lines", "12", "--app", "ashamed", "--follow"})
	source := flagset.Lookup("source")
	c.Check(source, check.NotNil)
	c.Check(source.Name, check.Equals, "source")
	c.Check(source.Usage, check.Equals, "The log from the given source")
	c.Check(source.Value.String(), check.Equals, "tsuru")
	c.Check(source.DefValue, check.Equals, "")
	ssource := flagset.Lookup("s")
	c.Check(ssource, check.NotNil)
	c.Check(ssource.Name, check.Equals, "s")
	c.Check(ssource.Usage, check.Equals, "The log from the given source")
	c.Check(ssource.Value.String(), check.Equals, "tsuru")
	c.Check(ssource.DefValue, check.Equals, "")
	unit := flagset.Lookup("unit")
	c.Check(unit, check.NotNil)
	c.Check(unit.Name, check.Equals, "unit")
	c.Check(unit.Usage, check.Equals, "The log from the given unit")
	c.Check(unit.Value.String(), check.Equals, "abcdef")
	c.Check(unit.DefValue, check.Equals, "")
	sunit := flagset.Lookup("u")
	c.Check(sunit, check.NotNil)
	c.Check(sunit.Name, check.Equals, "u")
	c.Check(sunit.Usage, check.Equals, "The log from the given unit")
	c.Check(sunit.Value.String(), check.Equals, "abcdef")
	c.Check(sunit.DefValue, check.Equals, "")
	lines := flagset.Lookup("lines")
	c.Check(lines, check.NotNil)
	c.Check(lines.Name, check.Equals, "lines")
	c.Check(lines.Usage, check.Equals, "The number of log lines to display")
	c.Check(lines.Value.String(), check.Equals, "12")
	c.Check(lines.DefValue, check.Equals, "10")
	slines := flagset.Lookup("l")
	c.Check(slines, check.NotNil)
	c.Check(slines.Name, check.Equals, "l")
	c.Check(slines.Usage, check.Equals, "The number of log lines to display")
	c.Check(slines.Value.String(), check.Equals, "12")
	c.Check(slines.DefValue, check.Equals, "10")
	app := flagset.Lookup("app")
	c.Check(app, check.NotNil)
	c.Check(app.Name, check.Equals, "app")
	c.Check(app.Usage, check.Equals, "The name of the app.")
	c.Check(app.Value.String(), check.Equals, "ashamed")
	c.Check(app.DefValue, check.Equals, "")
	sapp := flagset.Lookup("a")
	c.Check(sapp, check.NotNil)
	c.Check(sapp.Name, check.Equals, "a")
	c.Check(sapp.Usage, check.Equals, "The name of the app.")
	c.Check(sapp.Value.String(), check.Equals, "ashamed")
	c.Check(sapp.DefValue, check.Equals, "")
	follow := flagset.Lookup("follow")
	c.Check(follow, check.NotNil)
	c.Check(follow.Name, check.Equals, "follow")
	c.Check(follow.Usage, check.Equals, "Follow logs")
	c.Check(follow.Value.String(), check.Equals, "true")
	c.Check(follow.DefValue, check.Equals, "false")
	sfollow := flagset.Lookup("f")
	c.Check(sfollow, check.NotNil)
	c.Check(sfollow.Name, check.Equals, "f")
	c.Check(sfollow.Usage, check.Equals, "Follow logs")
	c.Check(sfollow.Value.String(), check.Equals, "true")
	c.Check(sfollow.DefValue, check.Equals, "false")
}
