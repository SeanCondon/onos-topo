// Copyright 2019-present Open Networking Foundation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import "github.com/spf13/cobra"

func getGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get {device} [args]",
		Short: "Get topology resources",
	}
	cmd.AddCommand(getGetDeviceCommand())
	return cmd
}

func getAddCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add {device} [args]",
		Short: "Add a topology resource",
	}
	cmd.AddCommand(getAddDeviceCommand())
	return cmd
}

func getUpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update {device} [args]",
		Short: "Update a topology resource",
	}
	cmd.AddCommand(getUpdateDeviceCommand())
	return cmd
}

func getRemoveCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove {device} [args]",
		Short: "Remove a topology resource",
	}
	cmd.AddCommand(getRemoveDeviceCommand())
	return cmd
}

func getWatchCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch {device} [args]",
		Short: "Watch for changes to a topology resource type",
	}
	cmd.AddCommand(getWatchDeviceCommand())
	return cmd
}
