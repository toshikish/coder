# Releases

Coder releases are cut directly from main in our [Github](https://github.com/coder/coder) on the first Thursday of each month.

We recommend enterprise customers test compatibility of new releases with their infrastructure on a staging environment before upgrading a production deployment.

We support two release channels: [mainline]() for the edge version of Coder and [stable]() for those with lower tolerance for fault. We field our mainline releases publicly for two weeks before promoting them to stable.

### Mainline releases
- Best suited for those with a configured staging environment
- Gives earliest access to latest features
- May include minor bugs
- All latest bugfixes and security patches are supported

### Stable releases
- Safest upgrade/installation path
- Best suited for large deployments
- May not include the latest features
- Security vulnerabilities and major bugfixes are supported

> Note: We support major security vulnerabilities (CVEs) for the past three versions of Coder.

## Installing stable

In general, we advise specifying the desired version when installing Coder from our releases page.

Best practices for installing Coder can be found on our [install](./index.md) pages.

<!--
<div class="tabs">

## Install script

By default, our install script points to the latest version of Coder, whether or not it has been elevated to stable.


```shell
curl -fsSL https://coder.com/install.sh | sh
```

Use the `--stable` flag to ensure installation of the latest stable release.

```shell
curl -fsSL https://coder.com/install.sh | sh -s -- --stable
```

## System Packages

## Helm

Whenever

</div> -->

## Release schedule


Release name | Date | Status
------------ | ---- | ------
2.5.0 | November 06, 2023 | Not supported
2.6.0 | December 06, 2023 | Not supported
2.7.0 | January 01, 2024 | Security Support
**2.8.0** | **Februrary 06, 2024** | **Stable**
2.9.0 | March 07, 2024 | Mainline
2.10.0 | April 04, 2024 | Not released
