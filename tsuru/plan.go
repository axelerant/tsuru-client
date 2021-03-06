// Copyright 2015 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	tsuruapp "github.com/tsuru/tsuru/app"
	"github.com/tsuru/tsuru/cmd"
	"launchpad.net/gnuflag"
)

type planList struct {
	human bool
	fs    *gnuflag.FlagSet
}

func (c *planList) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = gnuflag.NewFlagSet("plan-List", gnuflag.ExitOnError)
		human := "Humanized units for memory and swap."
		c.fs.BoolVar(&c.human, "human", false, human)
		c.fs.BoolVar(&c.human, "h", false, human)
	}
	return c.fs
}

func (c *planList) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "plan-list",
		Usage:   "plan-list [--human]",
		Desc:    "List available plans that can be used when creating an app.",
		MinArgs: 0,
	}
}

func renderPlans(plans []tsuruapp.Plan, isHuman bool) string {
	table := cmd.NewTable()
	table.Headers = []string{"Name", "Memory", "Swap", "Cpu Share", "Router", "Default"}
	for _, p := range plans {
		var memory, swap string
		if isHuman {
			memory = fmt.Sprintf("%d MB", p.Memory/1024/1024)
			swap = fmt.Sprintf("%d MB", p.Swap/1024/1024)
		} else {
			memory = fmt.Sprintf("%d", p.Memory)
			swap = fmt.Sprintf("%d", p.Swap)
		}
		table.AddRow([]string{
			p.Name, memory, swap,
			strconv.Itoa(p.CpuShare),
			p.Router,
			strconv.FormatBool(p.Default),
		})
	}
	return table.String()
}

func (c *planList) Run(context *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/plans")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	var plans []tsuruapp.Plan
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&plans)
	if err != nil {
		return err
	}
	if len(plans) == 0 {
		fmt.Fprintln(context.Stdout, "No plans available.")
		return nil
	}
	fmt.Fprintf(context.Stdout, "%s", renderPlans(plans, c.human))
	return nil
}

type appPlanChange struct {
	fs *gnuflag.FlagSet
	cmd.GuessingCommand
	cmd.ConfirmationCommand
}

func (c *appPlanChange) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "app-plan-change",
		Usage:   "app-plan-change <plan_name> [-a/--app appname] [-y/--assume-yes]",
		Desc:    "Change the plan of the application.",
		MinArgs: 1,
	}
}

func (c *appPlanChange) Run(context *cmd.Context, client *cmd.Client) error {
	context.RawOutput()
	appName, err := c.Guess()
	if err != nil {
		return err
	}
	plan := tsuruapp.Plan{Name: context.Args[0]}
	question := fmt.Sprintf("Are you sure you want to change the plan of the application %q to %q?", appName, plan.Name)
	if !c.Confirm(context, question) {
		return nil
	}
	url, err := cmd.GetURL(fmt.Sprintf("/apps/%s/plan", appName))
	if err != nil {
		return err
	}
	b, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	return cmd.StreamJSONResponse(context.Stdout, response)
}

func (c *appPlanChange) Flags() *gnuflag.FlagSet {
	if c.fs == nil {
		c.fs = cmd.MergeFlagSet(c.ConfirmationCommand.Flags(), c.GuessingCommand.Flags())
	}
	return c.fs
}
