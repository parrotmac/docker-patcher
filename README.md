# docker-patcher
`docker-patcher` provides a cli & API to calculate and apply binary diffs against Docker images. Diffing is done using `bsdiff`/`bspatch`, wrapped nicely by [@icedream/go-bsdiff](https://github.com/icedream/go-bsdiff).

While binary patching might not be particularly useful in traditional datacenter/server environments, it can be very beneficial for devices with limited internet connectivity (e.g. IoT/M2M).

# Warning: Work In Progress
The CLI has some rough edges. The package's main API was broken to make it easier to include in other projects.

# Usage

### Setup
```base
$ go build -o bin/didiff ./cmd/didiff
```

Extra: temporarily add these tools to your path
```bash
$ export PATH=$PATH:`pwd`/bin
```

### Create a patch
```bash
didiff create sha256:original_docker_image sha256:new_docker_image /path/to/diff.patch
```

### Apply a patch
```bash
didiff apply sha256:original_docker_image sha256:new_docker_image /path/to/diff.patch
```
`didiff apply` accepts an optional `-t` argument, specifying the a repo:tag to be applied to the new image, once loaded. This is optional.

# Example
In this example we'll create a patch to upgrade from `nginx:1.15.11` to `nginx:1.15.12`.

### Create a patch
To create the patch, ensure you have both the old and new images available:
```bash
$ docker images --no-trunc
REPOSITORY  TAG         IMAGE ID                                                                  CREATED       SIZE
nginx       1.15.11     sha256:bb776ce48575796501bcc53e511563116132b789ab0552d520513da8c738cba2   12 days ago   109MB
nginx       1.15.12     sha256:27a188018e1847b312022b02146bb7ac3da54e96fab838b7db9f102c8c3dd778   6 days ago    109MB
```

The shortened image IDs for 1.15.11 and 1.15.12 are `bb776ce48575` and `27a188018e18`, respectively.
```bash
didiff create bb776ce48575 27a188018e18 `pwd`/nginx_1-15-11_to_1-15-12.patch # Using full length IDs is supported, too
```

After a few seconds (or minutes, depending :)) the patch will be written to `nginx_1-15-1_to_1-15-12.patch`.

Upon inspecting the patch, we can see that in this case we're weighing in under 300KB! That's much better than a ~23MB pull (assuming caching).
```bash
$ du -hs ./nginx_1-15-11_to_1-15-12.patch
260K    ./nginx_1-15-11_to_1-15-12.patch
```

### Apply a patch
Applying the patch takes similar motions. If you're testing this out on a machine with the old and new images, you can easily remove an image with `docker rmi your_img_id_here`

Ensure the image doesn't exist locally:
```bash
$ docker images
REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
nginx               1.15.11             bb776ce48575        12 days ago         109MB
```

Now we can use the use the patch we built previously. Specifying `-t` ensures that the new image is properly re-tagged.
```bash
./didiff apply bb776ce48575 27a188018e18 `pwd`/nginx_1-15-11_to_1-15-12.patch -t nginx:1.15.12
```

We can see the image was patched and loaded by running `docker images`
```bash
$ docker images
REPOSITORY          TAG                 IMAGE ID            CREATED             SIZE
nginx               1.15.12             27a188018e18        6 days ago          109MB
nginx               1.15.11             bb776ce48575        12 days ago         109MB
```

As a final step, for good measure, let's verify the new image works!
```bash
$ docker run --rm -d \
    --name docker-patch-test \
    -p 8080:80 27a188018e18 \
    && curl -sSL -D - localhost:8080 -o /dev/null | grep Server: \
    && docker kill docker-patch-test
```
The above command should have printed (among other things!) `Server: nginx/1.15.12`, indicating success!

# Something is broken/could be better/etc...
Please feel free to open PRs, Issues, or send me an email!
