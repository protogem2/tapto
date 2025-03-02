/*
TapTo
Copyright (C) 2023, 2024 Callan Barrett

This file is part of TapTo.

TapTo is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

TapTo is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with TapTo.  If not, see <http://www.gnu.org/licenses/>.
*/

package launcher

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	s "strings"
	"time"

	"github.com/wizzomafizzo/mrext/pkg/input"

	"github.com/wizzomafizzo/mrext/pkg/games"
	mrextMister "github.com/wizzomafizzo/mrext/pkg/mister"
	"github.com/wizzomafizzo/tapto/pkg/config"
	"github.com/wizzomafizzo/tapto/pkg/platforms/mister"
)

func LaunchToken(cfg *config.UserConfig, manual bool, kbd input.Keyboard, text string) error {
	// detection can never be perfect, but these characters are illegal in
	// windows filenames and heavily avoided in linux. use them to mark that
	// this is a command
	if s.HasPrefix(text, "**") {
		text = s.TrimPrefix(text, "**")
		parts := s.SplitN(text, ":", 2)
		if len(parts) < 2 {
			return fmt.Errorf("invalid command: %s", text)
		}

		cmd, args := s.TrimSpace(parts[0]), s.TrimSpace(parts[1])

		// TODO: search game file
		// TODO: game file by hash

		switch cmd {
		case "system":
			if s.EqualFold(args, "menu") {
				return mrextMister.LaunchMenu()
			}

			system, err := games.LookupSystem(args)
			if err != nil {
				return err
			}

			return mrextMister.LaunchCore(mister.UserConfigToMrext(cfg), *system)
		case "command":
			if !manual {
				return fmt.Errorf("commands must be manually run")
			}

			command := exec.Command("bash", "-c", args)
			err := command.Start()
			if err != nil {
				return err
			}

			return nil
		case "random":
			if args == "" {
				return fmt.Errorf("no system specified")
			}

			if args == "all" {
				return mrextMister.LaunchRandomGame(mister.UserConfigToMrext(cfg), games.AllSystems())
			}

			// TODO: allow multiple systems
			system, err := games.LookupSystem(args)
			if err != nil {
				return err
			}

			return mrextMister.LaunchRandomGame(mister.UserConfigToMrext(cfg), []games.System{*system})
		case "ini":
			inis, err := mrextMister.GetAllMisterIni()
			if err != nil {
				return err
			}

			if len(inis) == 0 {
				return fmt.Errorf("no ini files found")
			}

			id, err := strconv.Atoi(args)
			if err != nil {
				return err
			}

			if id < 1 || id > len(inis) {
				return fmt.Errorf("ini id out of range: %d", id)
			}

			return mrextMister.SetActiveIni(id)
		case "get":
			go func() {
				_, _ = http.Get(args)
			}()
			return nil
		case "key":
			code, err := strconv.Atoi(args)
			if err != nil {
				return err
			}

			kbd.Press(code)

			return nil
		case "coinp1":
			amount, err := strconv.Atoi(args)
			if err != nil {
				return err
			}

			for i := 0; i < amount; i++ {
				kbd.Press(6)
				time.Sleep(100 * time.Millisecond)
			}

			return nil
		case "coinp2":
			// TODO: this is lazy, make a function
			amount, err := strconv.Atoi(args)
			if err != nil {
				return err
			}

			for i := 0; i < amount; i++ {
				kbd.Press(7)
				time.Sleep(100 * time.Millisecond)
			}

			return nil
		default:
			return fmt.Errorf("unknown command: %s", cmd)
		}
	}

	// if it's not a command, assume it's some kind of file path
	if filepath.IsAbs(text) {
		return mrextMister.LaunchGenericFile(mister.UserConfigToMrext(cfg), text)
	}

	// if it's a relative path with no extension, assume it's a core
	if filepath.Ext(text) == "" {
		return mrextMister.LaunchShortCore(text)
	}

	// if the file is in a .zip, just check .zip exists in each games folder
	parts := s.Split(text, "/")
	for i, part := range parts {
		if s.HasSuffix(s.ToLower(part), ".zip") {
			zipPath := filepath.Join(parts[:i+1]...)
			for _, folder := range games.GetGamesFolders(mister.UserConfigToMrext(cfg)) {
				if _, err := os.Stat(filepath.Join(folder, zipPath)); err == nil {
					return mrextMister.LaunchGenericFile(mister.UserConfigToMrext(cfg), filepath.Join(folder, text))
				}
			}
			break
		}
	}

	// then try check for the whole path in each game folder
	for _, folder := range games.GetGamesFolders(mister.UserConfigToMrext(cfg)) {
		path := filepath.Join(folder, text)
		if _, err := os.Stat(path); err == nil {
			return mrextMister.LaunchGenericFile(mister.UserConfigToMrext(cfg), path)
		}
	}

	return fmt.Errorf("could not find file: %s", text)
}
