/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package survey

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"

	sConfig "github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/output"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/timeutil"
)

const (
	Form = `Thank you for offering your feedback on Skaffold! Understanding your experiences and opinions helps us make Skaffold better for you and other users.

Skaffold will now attempt to open the survey in your default web browser. You may also manually open it using this URL:

%s

Tip: To permanently disable the survey prompt, run:
   skaffold config set --survey --global disable-prompt true`
)

var (
	// for testing
	isStdOut             = output.IsStdout
	open                 = browser.OpenURL
	updateSurveyPrompted = sConfig.UpdateGlobalSurveyPrompted
)

type Runner struct {
	configFile string
}

func New(configFile string) *Runner {
	return &Runner{
		configFile: configFile,
	}
}

func (s *Runner) ShouldDisplaySurveyPrompt() bool {
	cfg, disabled := isSurveyPromptDisabled(s.configFile)
	return !disabled && !recentlyPromptedOrTaken(cfg)
}

func isSurveyPromptDisabled(configfile string) (*sConfig.GlobalConfig, bool) {
	cfg, err := sConfig.ReadConfigFile(configfile)
	if err != nil {
		return nil, false
	}
	return cfg, cfg != nil && cfg.Global != nil &&
		cfg.Global.Survey != nil &&
		cfg.Global.Survey.DisablePrompt != nil &&
		*cfg.Global.Survey.DisablePrompt
}

func recentlyPromptedOrTaken(cfg *sConfig.GlobalConfig) bool {
	if cfg == nil || cfg.Global == nil || cfg.Global.Survey == nil {
		return false
	}
	return timeutil.LessThan(cfg.Global.Survey.LastTaken, 90*24*time.Hour) ||
		timeutil.LessThan(cfg.Global.Survey.LastPrompted, 10*24*time.Hour)
}

func (s *Runner) DisplaySurveyPrompt(out io.Writer) error {
	if !isStdOut(out) {
		return nil
	}
	output.Green.Fprintf(out, hats.prompt())
	return updateSurveyPrompted(s.configFile)
}

func (s *Runner) OpenSurveyForm(_ context.Context, out io.Writer, id string) error {
	sc, ok := getSurvey(id)
	if !ok {
		return fmt.Errorf("invalid survey id %q - please enter one of %s", id, validKeys())
	}
	_, err := fmt.Fprintln(out, fmt.Sprintf(Form, sc.URL))
	if err != nil {
		return err
	}
	if err := open(sc.URL); err != nil {
		logrus.Debugf("could not open url %s", sc.URL)
		return err
	}
	// Currently we will only update the global config survey taken
	// When prompting for the survey, we need to use the same field.
	return sConfig.UpdateHaTSSurveyTaken(s.configFile)
}
