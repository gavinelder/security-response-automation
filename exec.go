// Package exec is the entry point for security automation Cloud Functions.
package exec

// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/pubsub"
	"github.com/googlecloudplatform/threat-automation/clients"
	"github.com/googlecloudplatform/threat-automation/cloudfunctions"
	"github.com/googlecloudplatform/threat-automation/entities"
)

const (
	authFile = "credentials/auth.json"
)

// RevokeExternalGrantsFolders is the entry point for IAM revoker Cloud Function.
//
// This Cloud Function will be triggered when Event Threat Detection
// detects an anomalous IAM grant. Once triggered this function will
// attempt to revoke the external members added to the policy if they match the provided
// list of disallowed domains. Additionally this method will only remove members if the
// project they were added to is within the specified folders. This configuration allows
// you to take a remediation action only on specific members and folders. For example,
// you may have a folder "development" where users can experiment without strict policies.
// However in your "production" folder you may want to revoke any grants that ETD finds as
// long as they match the domains you specify.
//
// Permissions required
//
// By default the service account used can only revoke projects that are found within the
// folder ID specified within `action-revoke-member-folders.tf`.
func RevokeExternalGrantsFolders(ctx context.Context, m pubsub.Message) error {
	lg, err := clients.NewLogger(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize logger client: %q", err)
	}
	l := entities.NewLogger(lg)
	defer l.Close()

	l.Info("Initializing external members IAM removal")

	if os.Getenv("folder_ids") == "" {
		return fmt.Errorf("folder_ids environment variable not found")
	}
	if os.Getenv("disallowed") == "" {
		return fmt.Errorf("disallowed environment variable not found")
	}
	crm, err := clients.NewCloudResourceManager(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize cloud resource manager client: %q", err)
	}

	stg, err := clients.NewStorage(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize storage client: %q", err)
	}
	r := entities.NewResource(crm, stg)

	ids := strings.Split(os.Getenv("folder_ids"), ",")
	d := strings.Split(os.Getenv("disallowed"), ",")
	return cloudfunctions.RevokeExternalGrantsFolders(ctx, m, r, ids, d, l)
}

// SnapshotDisk is the entry point for the auto creation of GCE snapshots Cloud Function.
//
// This Cloud Function will respond to Event Threat Detection **bad IP** findings. Once a bad IP
// finding is received this Cloud Function will look for any existing disk snapshots for the
// affected instance. If there are recent snapshots then no action is taken. If we have not
// taken a snapshot recently, take a new snapshot for each disk within the instance.
//
// Permissions required
//
// By default the service account can only be used to create snapshots for the projects
// specified in `action-snaphot-disk.tf`
//
// TODO: Support assigning roles at the folder and organization level.
func SnapshotDisk(ctx context.Context, m pubsub.Message) error {
	lg, err := clients.NewLogger(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize logger client: %q", err)
	}
	l := entities.NewLogger(lg)
	defer l.Close()

	l.Info("Initializing snapshot generation")

	crm, err := clients.NewCloudResourceManager(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize cloud resource manager client: %q", err)
	}

	stg, err := clients.NewStorage(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize storage client: %q", err)
	}
	r := entities.NewResource(crm, stg)

	cs, err := clients.NewCompute(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize compute client: %q", err)
	}
	h := entities.NewHost(cs)

	return cloudfunctions.CreateSnapshot(ctx, m, r, h, l)

}

// CloseBucket will remove any public users from buckets found within the provided folders.
func CloseBucket(ctx context.Context, m pubsub.Message) error {
	lg, err := clients.NewLogger(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize logger client: %q", err)
	}
	l := entities.NewLogger(lg)
	defer l.Close()

	l.Info("Initializing bucket public users removal")

	if os.Getenv("folder_ids") == "" {
		return fmt.Errorf("folder_ids environment variable not found")
	}
	crm, err := clients.NewCloudResourceManager(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize cloud resource manager client: %q", err)
	}

	stg, err := clients.NewStorage(ctx, authFile)
	if err != nil {
		return fmt.Errorf("failed to initialize storage client: %q", err)
	}
	r := entities.NewResource(crm, stg)
	conf := &cloudfunctions.Configuration{FoldersIDs: strings.Split(os.Getenv("folder_ids"), ",")}
	return cloudfunctions.CloseBucket(ctx, m, r, l, conf)
}
