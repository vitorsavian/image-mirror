# image-mirror

Mirrors the third-party container images that K3s ships into `ghcr.io/k3s-io`,
so that installing K3s does not consume Docker Hub's anonymous pull quota.

The mirror is a byte-for-byte copy: same digests, same manifest lists, same
architectures. Only the registry host and the namespace of a reference change.

[`regsync.yaml`](regsync.yaml) is the list, and it is the whole repository —
that plus [regsync](https://github.com/regclient/regclient) and
[one workflow](.github/workflows/mirror.yml). There is no code here.

| Upstream | Mirror |
| --- | --- |
| `docker.io/coredns/coredns` | `ghcr.io/k3s-io/mirrored-coredns-coredns` |
| `docker.io/library/busybox` | `ghcr.io/k3s-io/mirrored-library-busybox` |
| `docker.io/library/traefik` | `ghcr.io/k3s-io/mirrored-library-traefik` |
| `registry.k8s.io/metrics-server/metrics-server` | `ghcr.io/k3s-io/mirrored-metrics-server` |
| `registry.k8s.io/pause` | `ghcr.io/k3s-io/mirrored-pause` |

The `mirrored-<repo>-<name>` names are the ones
[rancher/artifact-mirror](https://github.com/rancher/artifact-mirror) uses, so
K3s references keep the shape they already have.

## How it works

Every tag is written out, one entry each:

```yaml
- source: docker.io/coredns/coredns:1.14.7
  target: ghcr.io/k3s-io/mirrored-coredns-coredns:1.14.7
  type: image
```

Nothing is discovered and nothing arrives on its own. Reading `regsync.yaml`
tells you exactly what is in the mirror, and `regsync check` will tell you the
same thing from the registries' side.

For each entry regsync compares the manifest digest on both ends and, if they
differ, copies the manifest (or the multi-arch index and every manifest under
it) and mounts or uploads each blob. It does not unpack or rebuild anything, so
the bytes and the digest come out identical to upstream.

Three things follow from that and are worth knowing:

- **regsync never deletes.** Removing a line here does not remove the tag from
  GHCR. Cleanup is by hand.
- **Copying is idempotent.** A run with nothing to do is a digest comparison per
  entry and finishes in well under a second, so running the workflow when in
  doubt is free.
- **The mirror follows upstream.** If a tag is rebuilt upstream — official
  images are, regularly — the next run notices the digest changed and re-copies
  it. That is the point of comparing rather than passing `--missing`, which
  skips any tag the target already has without looking at it, and would leave
  the mirror serving the old bytes indefinitely. To freeze a specific image
  instead, pin its source by digest rather than by tag.

## When the mirror runs

- a push to `master` that touches `regsync.yaml` — a merged pull request
  mirrors what it added;
- **Run workflow** on the Actions tab, to retry after a failure.

There is no schedule, and with an explicit list there is nothing for one to do:
the mirror can only change when this file changes.

## Adding a version

Add the three lines and open a pull request. The check on the pull request runs
`regsync check`, which fails if the tag does not exist upstream — a typo is
caught there, not discovered later by a K3s build.

Keep entries that older K3s branches still reference. The list is the union
across supported branches, not just what `main` uses, so a bump normally adds a
line and removes the old one only once every branch has moved.

### The mirror has to lead K3s

K3s' airgap test pulls every image in `scripts/airgap/image-list.txt` on the
bump pull request, before it merges. A tag the mirror does not have makes that
pull request fail. So, before opening a bump in K3s:

1. add the tag here and merge it — merging runs the mirror;
2. confirm the tag is in GHCR;
3. then open the K3s pull request.

## New packages start private

GHCR creates a package the first time something is pushed to it, and it creates
it **private**. Nothing in a workflow can change that; somebody with org
permissions has to flip it to public under
[the org's packages](https://github.com/orgs/k3s-io/packages).

So the first mirror run for a new image succeeds and the image is still
unpullable. The last step of the mirror job catches this: it asks GHCR's token
endpoint for an anonymous pull token per package and fails on anything that
does not answer `200`. It cannot run at pull-request time — anonymously, a
private package and a package that does not exist both answer `403`, so there
is nothing to distinguish before the push happens.

Pushing needs nothing configured: the workflow's own `GITHUB_TOKEN` creates the
package and is granted access to it. That only holds if the name is free — a
package that already exists under the org and belongs to a different repository
rejects the push with a 403.

Mirrored images keep whatever labels upstream set, including
`org.opencontainers.image.source`. For traefik that points at
`github.com/traefik/traefik`, which is not in this org, so GHCR leaves the
package with no linked repository. That is cosmetic; relabelling it would
change the digest and cost the byte-for-byte property, so it is left alone.

## Running it locally

```sh
curl -fsSL -o regsync \
  https://github.com/regclient/regclient/releases/download/v0.11.6/regsync-linux-amd64
chmod +x regsync

./regsync --config regsync.yaml config   # does it parse
./regsync --config regsync.yaml check    # what would be copied
```

`check` needs no credentials to read the sources. To try a real copy without
touching GHCR, point the targets at a local registry:

```sh
docker run -d --rm -p 5555:5000 --name regsync-test registry:2
sed -e 's|ghcr\.io|localhost:5555|' \
    -e 's|^    repoAuth: true$|    tls: disabled|' regsync.yaml > /tmp/local.yaml
./regsync --config /tmp/local.yaml once
```

If you are pulling Docker Hub anonymously and hit `TOOMANYREQUESTS`, just run it
again — tags already copied are skipped after a digest check. Adding a `docker.io`
entry to `creds` with a Docker Hub login raises the limit, but at three tags from
Docker Hub it has not been needed.

## Trying it in a fork

Both workflows rewrite the targets to the namespace of whoever owns the
repository before running regsync. Here that is `k3s-io` and the step changes
nothing; in a fork it aims everything at the fork's own packages, so pushing,
package creation and the visibility check all get exercised for real without
touching `k3s-io`. `regsync.yaml` stays literal, and the rewritten copy
(`sync.yaml`) is never committed.

The only thing that needs setting up is the token: a fork's `GITHUB_TOKEN`
already has `packages: write`, but pushing workflow files from the command line
needs `gh auth refresh -h github.com -s workflow`.

After the first run the packages are private, so the visibility step fails on
purpose. It names each one and prints the link to your packages page; make them
public there and re-run the workflow.
