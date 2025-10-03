package endpoints

import (
	"context"
	"os/exec"
	"time"

	"github.com/cordialsys/panel/pkg/client"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/gofiber/fiber/v2"
)

// hardcode which services are exposed to the API
type serviceDescription struct {
	Name string `json:"name"`
	// not writable from the API
	ReadOnly bool `json:"read_only"`
	Logs     bool `json:"logs"`
}
type serviceDescriptions []serviceDescription

func (s serviceDescriptions) canRead(name string) bool {
	for _, srv := range s {
		if srv.Name == name {
			return true
		}
	}
	return false
}

func (s serviceDescriptions) canReadLogs(name string) bool {
	for _, srv := range s {
		if srv.Name == name {
			return srv.Logs
		}
	}
	return false
}

func (s serviceDescriptions) canWrite(name string) bool {
	for _, srv := range s {
		if srv.Name == name {
			return !srv.ReadOnly
		}
	}
	return false
}

const ServiceTreasury = "treasury.service"
const ServiceStartTreasury = "start-treasury.service"
const ServiceBlueprint = "blueprint.service"

// These should match the systemd services on the host that we
// want to expose to the API.
var SERVICES = serviceDescriptions{
	{
		Name: "docker.socket",
	},
	{
		Name: "docker.service",
	},
	{
		Name: ServiceTreasury,
		Logs: true,
	},
	{
		Name: ServiceStartTreasury,
		Logs: true,
	},
	{
		Name: "treasury-firewall.service",
		Logs: true,
	},
	{
		Name: ServiceBlueprint,
		Logs: true,
	},
	{
		Name:     "panel.service",
		ReadOnly: true,
	},
}

func newService(unit dbus.UnitStatus) client.Service {
	return client.Service{
		Name:        unit.Name,
		Description: unit.Description,
		LoadState:   unit.LoadState,
		ActiveState: client.ServiceState(unit.ActiveState),
		SubState:    unit.SubState,
		JobType:     unit.JobType,
	}
}

func getSystemdService(ctx context.Context, serviceName string) (client.Service, error) {
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return client.Service{}, servererrors.InternalErrorf("failed to connect to systemd: %v", err)
	}
	units, err := conn.ListUnitsByNamesContext(ctx, []string{serviceName})
	if err != nil {
		return client.Service{}, servererrors.InternalErrorf("failed to get units: %v", err)
	}
	if len(units) == 0 {
		return client.Service{}, servererrors.NotFoundf("service %s not found", serviceName)
	}

	srv := newService(units[0])
	return srv, nil
}

func (endpoints *Endpoints) GetService(c *fiber.Ctx) error {
	ctx := c.Context()
	serviceName := c.Params("service")
	if serviceName == "" {
		return servererrors.BadRequestf("service name parameter missing in path")
	}
	if !SERVICES.canRead(serviceName) {
		return servererrors.BadRequestf("service %s not found", serviceName)
	}
	srv, err := getSystemdService(ctx, serviceName)
	if err != nil {
		return err
	}

	return c.JSON(srv)
}

func (endpoints *Endpoints) GetServiceLogs(c *fiber.Ctx) error {
	serviceName := c.Params("service")
	if serviceName == "" {
		return servererrors.BadRequestf("service name parameter missing in path")
	}
	if !SERVICES.canReadLogs(serviceName) {
		return servererrors.BadRequestf("service %s not found", serviceName)
	}

	cmd := exec.Command("journalctl", "-u", serviceName, "-n", "1000", "--no-pager", "-o", "cat")
	bz, err := cmd.CombinedOutput()
	if err != nil {
		return servererrors.InternalErrorf("failed to read %s logs: %v", serviceName, err)
	}
	return c.SendString(string(bz))
}

func (endpoints *Endpoints) GetContainerLogs(c *fiber.Ctx) error {
	serviceName := c.Params("container")
	if serviceName == "" {
		return servererrors.BadRequestf("container name parameter missing in path")
	}
	switch serviceName {
	case "treasury":
		// ok
	default:
		return servererrors.BadRequestf("unsupported container: %s", serviceName)
	}

	cmd := exec.Command("docker", "logs", "--tail", "1000", serviceName)
	bz, err := cmd.CombinedOutput()
	if err != nil {
		return servererrors.InternalErrorf("failed to read %s logs: %v", serviceName, err)
	}
	return c.SendString(string(bz))
}

func (endpoints *Endpoints) ListServices(c *fiber.Ctx) error {
	ctx := c.Context()
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return servererrors.InternalErrorf("failed to connect to systemd: %v", err)
	}

	srvs := []client.Service{}
	for _, srv := range SERVICES {
		units, err := conn.ListUnitsByNamesContext(ctx, []string{srv.Name})
		if err != nil {
			return servererrors.InternalErrorf("failed to get units: %v", err)
		}
		// Note that this systemd API seems to ALWAYS return a value, for any service name,
		if len(units) == 0 {
			// This is what systemd always returns, we just maintain it in case it changes:
			srvs = append(srvs, client.Service{
				Name:        srv.Name,
				Description: srv.Name,
				LoadState:   "not-found",
				ActiveState: "inactive",
				SubState:    "dead",
			})
		} else {
			srvs = append(srvs, newService(units[0]))
		}
	}

	return c.JSON(srvs)
}

type ServiceAction string

const ServiceActionStart ServiceAction = "start"
const ServiceActionStop ServiceAction = "stop"
const ServiceActionRestart ServiceAction = "restart"
const ServiceActionDisable ServiceAction = "disable"
const ServiceActionEnable ServiceAction = "enable"

func updateSystemdService(ctx context.Context, serviceName string, action ServiceAction) (*client.Service, error) {
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return nil, servererrors.InternalErrorf("failed to connect to systemd: %v", err)
	}
	switch action {
	case ServiceActionStart:
		_, err = conn.StartUnitContext(ctx, serviceName, "replace", nil)
	case ServiceActionStop:
		_, err = conn.StopUnitContext(ctx, serviceName, "replace", nil)
	case ServiceActionRestart:
		_, err = conn.RestartUnitContext(ctx, serviceName, "replace", nil)
	case ServiceActionDisable:
		_, err = conn.DisableUnitFilesContext(ctx, []string{serviceName}, false)
	case ServiceActionEnable:
		_, _, err = conn.EnableUnitFilesContext(ctx, []string{serviceName}, false, false)
	default:
		return nil, servererrors.BadRequestf("unknown action: %s", action)
	}
	if err != nil {
		return nil, servererrors.InternalErrorf("failed to update service: %v", err)
	}
	units, err := conn.ListUnitsByNamesContext(ctx, []string{serviceName})
	if err != nil {
		return nil, servererrors.InternalErrorf("failed to get units after executing action: %v", err)
	}
	if len(units) == 0 {
		return nil, servererrors.NotFoundf("service %s not found after executing action", serviceName)
	}
	srv := newService(units[0])
	return &srv, nil
}

const DefaultStopTimeout = 30 * time.Second

// Stop a systemd service and wait for it to stop
func stopSystemdServiceAndWait(ctx context.Context, serviceName string) (didIssueStop bool, err error) {
	treasury, _ := getSystemdService(ctx, serviceName)
	if treasury.ActiveState == client.ServiceStateInactive || treasury.ActiveState == client.ServiceStateFailed {
		return didIssueStop, nil
	}
	updateSystemdService(ctx, serviceName, "stop")
	didIssueStop = true
	err = waitSystemdServiceToStop(ctx, serviceName, DefaultStopTimeout)
	return didIssueStop, err
}

func waitSystemdServiceToStop(ctx context.Context, serviceName string, timeout time.Duration) (err error) {
	start := time.Now()
	for {
		treasury, _ := getSystemdService(ctx, serviceName)
		if treasury.ActiveState != "active" &&
			treasury.ActiveState != "deactivating" &&
			treasury.ActiveState != "activating" {
			return nil
		}
		time.Sleep(1 * time.Second)
		if time.Since(start) > timeout {
			return servererrors.InternalErrorf("%s did not stop after %s", serviceName, time.Since(start))
		}
	}
}

func (endpoints *Endpoints) UpdateService(c *fiber.Ctx) error {
	ctx := c.Context()

	serviceName := c.Params("service")
	if serviceName == "" {
		return servererrors.BadRequestf("service name parameter missing in path")
	}
	action := c.Params("action")
	if action == "" {
		return servererrors.BadRequestf("action parameter missing in path")
	}
	if !SERVICES.canWrite(serviceName) {
		return servererrors.BadRequestf("service %s not found", serviceName)
	}

	srv, err := updateSystemdService(ctx, serviceName, ServiceAction(action))
	if err != nil {
		return err
	}

	return c.JSON(srv)
}
