/***************************************************************
 *
 * Copyright (C) 2024, Pelican Project, Morgridge Institute for Research
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you
 * may not use this file except in compliance with the License.  You may
 * obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 ***************************************************************/
package config

import (
	"reflect"
	"testing"
)

func TestApplyOverrides(t *testing.T) {
	// Define default config values
	defaultConfig := &Config{
		LogLevel: "INFO",
		ConsoleOutput: ConsoleOutputConfig{
			Enabled:    true,
			JSONOutput: false,
			Colors:     true,
		},
		FileOutput: FileOutputConfig{
			Enabled:     true,
			FilePath:    "/var/log/chtc/app.log",
			MaxFileSize: 100,
			MaxBackups:  5,
			MaxAgeDays:  30,
		},
	}

	// Define override values
	overrides := &Config{
		LogLevel: "DEBUG",
		FileOutput: FileOutputConfig{
			FilePath:    "/custom/path/logfile.log",
			MaxFileSize: 200,
		},
	}

	// Expected config after applying overrides
	expectedConfig := &Config{
		LogLevel: "DEBUG", // Overridden
		ConsoleOutput: ConsoleOutputConfig{
			Enabled:    true,  // Default retained
			JSONOutput: false, // Default retained
			Colors:     true,  // Default retained
		},
		FileOutput: FileOutputConfig{
			Enabled:     true,                       // Default retained
			FilePath:    "/custom/path/logfile.log", // Overridden
			MaxFileSize: 200,                        // Overridden
			MaxBackups:  5,                          // Default retained
			MaxAgeDays:  30,                         // Default retained
		},
	}

	// Apply overrides
	ApplyOverrides(defaultConfig, overrides)

	// Verify the results
	if !reflect.DeepEqual(defaultConfig, expectedConfig) {
		t.Errorf("ApplyOverrides failed.\nGot: %+v\nWant: %+v", defaultConfig, expectedConfig)
	}
}
