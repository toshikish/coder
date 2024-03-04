# Releases 

Coder releases are cut directly from main in our [Github](https://github.com/coder/coder) on the first Thursday of each month.

We reccomend enterprise customers test compatibility of new releases with their infrastructure on a staging environment before upgrading a production deployment. 

Since not all customers have a staging environment, we label our latest gauranteed-stable release tag with, implying it has been thouroughly vetted internally and by our community. This `stable` indicator is applied once the latest release has been in flight for at least 2 weeks, normally on the third Thursday of each month. 

### Latest releases
- Best suited for those with a configured staging environment
- Gives earliest access to latest features
- May include minor bugs (when not promoted to stable)
- All latest bugfixes and security patches are supported

### Stable releases
- Safest upgrade/installation path
- Best suited for large deployments
- May not include the latest features
- Security vulnerabilities and major bugfixes are supported 

> Note: We support major security vulnerabilities (CVEs) for the past three versions of Coder.

## Installing stable

Best practices for installing coder can be found on our [install](./index.md) pages. 

<!-- 
<div class="tabs">

## Install script

By default, our install script points to the latest version of Coder, whether or not it has been elevated to stable.


```shell
curl -fsSL https://coder.com/install.sh | sh
```

Use the `--stable` flag to ensure installation of the latest stable release.

```shell
curl -fsSL https://coder.com/install.sh | sh --stable
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
**2.9.0** | **March 07, 2024** | **Latest**
2.10.0 | April 04, 2024 | Not released
