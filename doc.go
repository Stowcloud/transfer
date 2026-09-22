// Package transfer defines transport- and storage-neutral contracts for
// receiving, staging, publishing, and reconciling content transfers.
//
// It deliberately does not authorize callers, open files, issue network
// requests, or select a provider. Those decisions belong to the application
// and the strategy implementation that it supplies.
package transfer
