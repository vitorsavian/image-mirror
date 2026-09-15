<!--
Most pull requests here add a tag to regsync.yaml because K3s is moving to it.
-->

#### What this changes

<!-- Which image, from which version to which. -->

#### Why

<!-- Usually: K3s is bumping to this version. Link the K3s issue or PR. -->

#### Checklist

- [ ] `regsync check` passed on this PR, so the tag exists upstream.
- [ ] I kept the versions older K3s branches still reference, and only removed
      a line if no supported branch uses it any more.
- [ ] If this adds a new image: the package will be private after the first
      mirror run, and somebody with org permissions has to make it public
      before K3s can pull it. The mirror job's visibility step fails until then.
- [ ] Merging this runs the mirror. The K3s bump PR comes after, not before.
