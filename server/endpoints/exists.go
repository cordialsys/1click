package endpoints

import (
	"os"
	"time"

	"github.com/cordialsys/panel/pkg/plog"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

type FileInfo struct {
	Mode    string    `json:"mode"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Dir     bool      `json:"dir"`
}

func (endpoints *Endpoints) Stat(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return servererrors.BadRequestf("path query parameter is required")
	}
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return servererrors.NotFoundf("path '%s' does not exist", path)
	}
	if err != nil {
		return servererrors.InternalErrorf("%v", err)
	}

	info := &FileInfo{
		Mode:    stat.Mode().String(),
		Size:    stat.Size(),
		ModTime: stat.ModTime(),
		Dir:     stat.IsDir(),
	}

	return c.JSON(info)
}

func (endpoints *Endpoints) Ls(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return servererrors.BadRequestf("path query parameter is required")
	}
	files, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return servererrors.NotFoundf("path '%s' does not exist", path)
	}
	if err != nil {
		return servererrors.InternalErrorf("%v", err)
	}
	names := make([]string, len(files))
	for i, file := range files {
		names[i] = file.Name()
	}

	return c.JSON(names)
}

func (endpoints *Endpoints) Logs(c *fiber.Ctx) error {
	return c.JSON(plog.LogHistory)
}
