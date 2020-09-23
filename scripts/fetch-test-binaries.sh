#!/usr/bin/env bash
curl https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-1.15.5-${1}-amd64.tar.gz -o kubebuilder-tools.tar.gz
tar -zvxf kubebuilder-tools.tar.gz

rm kubebuilder-tools.tar.gz