package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/venlax/c_build/internal/config"
)

var Cli *client.Client = nil
var containerID string = ""

func Init(create bool) {
	ctx := context.Background()
	var err error
	Cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}

	if _, err := Cli.ImageInspect(ctx, config.Image); err == nil {
		fmt.Fprintf(os.Stdout, "use local docker image %s\n", config.Image)
	} else {
		readCloser, err := Cli.ImagePull(ctx, config.Image, image.PullOptions{})
		if err != nil {
			panic(err)
		}
		defer readCloser.Close()

		_, err = io.Copy(os.Stdout, readCloser)
		if err != nil {
			panic(err)
		}
	}

	if create {
		resp, err := Cli.ContainerCreate(ctx, &container.Config{
			Image:      config.Image,
			Cmd:        []string{"tail", "-f", "/dev/null"},
			WorkingDir: config.WorkingDir,
			Env:        config.Env,
		}, &container.HostConfig{
			Privileged: true,
			Binds: []string{
				config.HostBuildRootDir + ":" + config.WorkingDir,
				config.HostReprobuildDir + ":" + config.ReprobuildDir,
				"/sys:/sys",
				"/lib/modules:/lib/modules",
				"/usr/src:/usr/src",
			},
			PidMode:     "host",
			NetworkMode: "host",
		}, nil, nil, config.ContainerName)

		if err != nil {
			panic(err)
		}
		containerID = resp.ID

		if err := Cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
			panic(err)
		}
	} else {
		if ok, c := getContainer(config.ContainerName); ok {
			containerID = c.ID
			inspect := getContainerInspect()
			if !inspect.State.Running {
				if err := Cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
					panic(err)
				}
			}
		} else {
			panic(fmt.Sprintf("No container named <%s>, try rerun with args -c/--create", config.ContainerName))
		}
	}

	// inspect, err := Cli.ContainerInspect(ctx, containerID)
	// fmt.Println(inspect.State.Status)
	// if inspect.State.Running {
	// 	fmt.Println("Running")
	// }

}

func Run(command []string, writer io.Writer) error {
	ctx := context.Background()

	resp, err := Cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          command,
		AttachStdout: true,
		AttachStderr: true,
	})

	if err != nil {
		return err
	}

	hijacked_resp, err := Cli.ContainerExecAttach(ctx, resp.ID, container.ExecAttachOptions{})

	if err != nil {
		return err
	}

	defer hijacked_resp.Close()

	// _, err = io.Copy(os.Stdout, hijacked_resp.Reader)
	// _, err = io.Copy(writer, hijacked_resp.Reader)
	_, err = stdcopy.StdCopy(writer, os.Stderr, hijacked_resp.Reader)

	if err != nil {
		return err
	}

	inspect, err := Cli.ContainerExecInspect(ctx, resp.ID)

	if err != nil {
		return err
	}
	if inspect.ExitCode != 0 {
		return fmt.Errorf("exec failed, exit code=%d", inspect.ExitCode)
	}

	return nil
}

// func containerExist(name string) bool {
// 	containers, err := Cli.ContainerList(context.Background(), container.ListOptions{})
// 	if err != nil {
// 		panic(err)
// 	}

// 	return slices.ContainsFunc(containers, func(c container.Summary) bool {return c.Names[0][1:] == name})
// }

func getContainer(name string) (bool, container.Summary) {
	containers, err := Cli.ContainerList(context.Background(), container.ListOptions{})
	if err != nil {
		panic(err)
	}

	idx := slices.IndexFunc(containers, func(c container.Summary) bool { return c.Names[0][1:] == name })

	if idx == -1 {
		return false, container.Summary{}
	}
	return true, containers[idx]
}

func getContainerInspect() container.InspectResponse {
	inspect, err := Cli.ContainerInspect(context.Background(), containerID)
	if err != nil {
		panic(err)
	}
	return inspect
}

func GetImageInspect(imageID string) image.InspectResponse {
	inspect, err := Cli.ImageInspect(context.Background(), imageID)
	if err != nil {
		panic(err)
	}
	return inspect
}

func FileExists(filePath string) (bool, error) {
	_, err := Cli.ContainerStatPath(context.Background(), containerID, filePath)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}
