// Copyright (C) 2018 Storj Labs, Inc.
// See LICENSE for copying information.

package cmd

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/minio/minio/pkg/hash"
	"github.com/spf13/cobra"
	"github.com/zeebo/errs"

	"storj.io/storj/pkg/cfgstruct"
	"storj.io/storj/pkg/process"
)

var (
	cpCfg Config
	cpCmd = &cobra.Command{
		Use:   "cp",
		Short: "A brief description of your command",
		RunE:  copy,
	}
)

func init() {
	RootCmd.AddCommand(cpCmd)
	cfgstruct.Bind(cpCmd.Flags(), &cpCfg, cfgstruct.ConfDir(defaultConfDir))
	cpCmd.Flags().String("config", filepath.Join(defaultConfDir, "config.yaml"), "path to configuration")
}

func copy(cmd *cobra.Command, args []string) (err error) {
	ctx := process.Ctx(cmd)

	if len(args) == 0 {
		return errs.New("No file specified for copy")
	}

	if len(args) == 1 {
		return errs.New("No destination specified")
	}

	storjObjects, err := getStorjObjects(ctx, cpCfg)
	if err != nil {
		return err
	}

	sourceFile, err := os.Open(args[0])
	if err != nil {
		return err
	}

	fileInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	fileReader, err := hash.NewReader(sourceFile, fileInfo.Size(), "", "")
	if err != nil {
		return err
	}

	defer sourceFile.Close()

	destFile, err := url.Parse(args[1])
	if err != nil {
		return err
	}

	objInfo, err := storjObjects.PutObject(ctx, destFile.Host, destFile.Path, fileReader, nil)
	if err != nil {
		return err
	}

	fmt.Println("Bucket:", objInfo.Bucket)
	fmt.Println("Object:", objInfo.Name)

	return nil
}
