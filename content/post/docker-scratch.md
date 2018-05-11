---
title: "Running Untrusted Code in a Secure Docker Container from Scratch"
date: 2018-05-10T16:18:25-07:00
---

As part of [Luk.ai](https://luk.ai/) we need to be able to run Tensorflow within
a secure environment since a running Tensorflow model can do pretty much
anything it wants to the host system.

For ease of deployment, we'd also like to be able to use Docker since it
provides nice sandboxing support and ability to limit resources used by the
container. We'd also like for the container to not be able to do anything other
than run models. Thus, we needed a completely stripped down container with no
permissions set. It's also pretty nice if it's as small as possible.

## Building A Minimal Container

Thus enters building a Dockerfile with just the minimum dependencies.

Normally you can just build a Go binary without any shared dependencies by running:
```bash
$ go build -v -tags netgo -installsuffix netgo -ldflags '-w -s' .
```

However, since this container needs to be able to run Tensorflow we need to be
able to link against the `libtensorflow.so` file and it's dependencies. To find
all the dependencies we can use `ldd`.

```
$ ldd secagg
        linux-vdso.so.1 (0x00007ffcbd1c3000)
        libtensorflow.so => /usr/lib/libtensorflow.so (0x00007f007ab54000)
        libpthread.so.0 => /usr/lib/libpthread.so.0 (0x00007f007a936000)
        libc.so.6 => /usr/lib/libc.so.6 (0x00007f007a57a000)
        libtensorflow_framework.so => /usr/lib/libtensorflow_framework.so (0x00007f0079590000)
        libcublas.so.9.1 => /opt/cuda/lib64/libcublas.so.9.1 (0x00007f0075e6c000)
        libcusolver.so.9.1 => /opt/cuda/lib64/libcusolver.so.9.1 (0x00007f00706f7000)
        libcudart.so.9.1 => /opt/cuda/lib64/libcudart.so.9.1 (0x00007f0070489000)
        libdl.so.2 => /usr/lib/libdl.so.2 (0x00007f0070285000)
        libgomp.so.1 => /usr/lib/libgomp.so.1 (0x00007f0070057000)
        libm.so.6 => /usr/lib/libm.so.6 (0x00007f006fcc2000)
        libstdc++.so.6 => /usr/lib/libstdc++.so.6 (0x00007f006f939000)
        libgcc_s.so.1 => /usr/lib/libgcc_s.so.1 (0x00007f006f721000)
        /lib64/ld-linux-x86-64.so.2 => /usr/lib64/ld-linux-x86-64.so.2 (0x00007f00a0484000)
        libcuda.so.1 => /usr/lib/libcuda.so.1 (0x00007f006eb81000)
        libcudnn.so.7 => /opt/cuda/lib64/libcudnn.so.7 (0x00007f005a31e000)
        libcufft.so.9.1 => /opt/cuda/lib64/libcufft.so.9.1 (0x00007f0052e31000)
        libcurand.so.9.1 => /opt/cuda/lib64/libcurand.so.9.1 (0x00007f004eeae000)
        librt.so.1 => /usr/lib/librt.so.1 (0x00007f004eca6000)
        libnvidia-fatbinaryloader.so.390.48 => /usr/lib/libnvidia-fatbinaryloader.so.390.48 (0x00007f004ea5a000
```

The `linux-vdso.so.1` file isn't a real dependency and is automatically injected
by the Linux kernel, the rest however we need to include in the container.

There's a little bit of bash code used to copy all of those files into a
directory `root/`.

```bash
mkdir -p root
cd root
for f in $(ldd ../secagg | sed -n 's/.*\s\(\/.*\) .*/\1/p'); do
  cp --parents "$f" .
done
cd ..
```

There's still one other requirement to get a running system and that's a
`/etc/passwd` file for the `nobody` user.

passwd.minimal

```
nobody:x:65534:65534:Nobody:/:
```

Once we have all that, we can now create the Dockerfile and build the container.

```Dockerfile
FROM scratch
ENV LD_LIBRARY_PATH /usr/local/lib:/usr/lib:/lib:/opt/cuda/lib64:/usr/lib64:/lib64
ADD root /
ADD passwd.minimal /etc/passwd
ADD secagg /secagg
USER nobody
CMD ["/usr/lib64/ld-linux-x86-64.so.2", "/secagg"]
```

## LD_LIBRARY_PATH

Docker will automatically detect shared files under `/lib` and `/lib64` but if
they're anywhere else we need to tell the system where to load them from. You
can do this by setting the `LD_LIBRARY_PATH` environment variable and then using
`ld.so` to run the binary.

The LD_LIBRARY_PATH above is what we needed to run the container on Arch Linux,
but it'll depend on what operating system it's being built on.

If you get a docker error like `standard_init_linux.go:190: exec user process
caused "no such file or directory"` it's likely a dynamic linking issue.

## Running the Container

To actually run the container, we launch it from the Go host process via the
Docker command line.

```go
ctx, cancel := context.WithCancel(context.TODO())
cmd := exec.CommandContext(
	ctx,
	"docker", "run",
	fmt.Sprintf("--name=%s", container),
	// disable network access
	"--net=none",
	// drop all process capabilities
	"--cap-drop=all",
	// attach to STDIN, STDOUT, STDERR
	"--attach=STDIN",
	"--attach=STDOUT",
	"--attach=STDERR",
	// allow sending and receiving on STDIN
	"-i",
	// limit CPU usage to 1 core
	"--cpus=1",
	// remove container after shutting down
	"--rm",
	// make the root file system read only
	"--read-only",
	// setup a /tmp directory without execute permission and limit to 1GB
	"--tmpfs=/tmp:rw,size=1g,mode=1777,noexec",
	"secagg:latest",
)
cmd.Stderr = os.Stderr
in, err := cmd.StdinPipe()
if err != nil {
	return nil, err
}
out, err := cmd.StdoutPipe()
if err != nil {
	return nil, err
}
if err := cmd.Start(); err != nil {
  return nil, err
}
```

## Communicating with the Container

Since the container is completely locked down with no network access we use the
`stdin` and `stdout` for communication. Logs are sent over `stderr` like normal.

Since we like having strongly typed messages, we use GRPC on both sides along
with `muxado` so we can have multiple connections to the container running over
the same ReadWriteCloser.

### Client/Host Side

```go
var _ io.ReadWriteCloser = RWC{}

var StdIORW = RWC{
	ReadCloser:  os.Stdin,
	WriteCloser: os.Stdout,
}

type RWC struct {
	io.ReadCloser
	io.WriteCloser
}

func (rw RWC) Close() error {
	if err := rw.WriteCloser.Close(); err != nil {
		return err
	}
	if err := rw.ReadCloser.Close(); err != nil {
		return err
	}
	return nil
}

// code to setup the connection over the stdin/stdout writer and reader.
rwc := seclib.RWC{
	ReadCloser:  out,
	WriteCloser: in,
}
mux := muxado.Client(rwc, nil)
conn, err := grpc.Dial(
	"",
	grpc.WithDialer(func(addr string, dur time.Duration) (gonet.Conn, error) {
		return mux.Open()
	}),
	grpc.WithInsecure(),
)
```

### Server/Container Side

```go
func main() {
	log.SetPrefix("[SecAgg] ")
	log.SetOutput(os.Stderr)
	log.SetFlags(log.Flags() | log.Lshortfile)

	log.Println("Running secure aggregator!")

	s := newRPCServer()
	grpcServer := grpc.NewServer(logging.GRPCCallLogger()...)
	secaggpb.RegisterAggServer(grpcServer, s)
	mux := muxado.Server(StdIORW, nil)
	defer mux.Close()
	if err := grpcServer.Serve(mux); err != nil {
		log.Fatal(err)
	}
	log.Println("Done!")
}
```
