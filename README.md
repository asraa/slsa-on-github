# slsa-on-github
Achieving SLSA 2+ on GitHub


Example usage
```
name: My Workflow
on: [push, pull_request]
jobs:
  build:
    steps:
      # Add build steps here
      # Upload it to use in the next step
      - name: Upload build
        uses: 'actions/upload-artifact@v2'
        with:
          name: main
          path: main
  provenance:
    runs-on: ubuntu-latest
    steps:
    - uses: asraa/slsa-on-github@master
      with:
        digest: $(sha256sum main | awk '{print $1}')
        repository: ${{ github.repository }}
```