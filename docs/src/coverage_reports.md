# Coverage Reports

This chapter will walk you through how to maximize coverage reports for your tests.

## What is a coverage report

A coverage report is a document or summary that provides information about how much of a codebase is covered by tests. It measures the extent to which the code is executed when the test suite runs, typically expressed as a percentage. The goal of a coverage report is to give developers insights into how thoroughly the code is tested, helping identify areas that might need additional testing to ensure reliability and reduce bugs.

## Setting up coverage report for medusa

In order to get a coverage report when running medusa tests, you need to set the corpusDirectory in `medusa`'s project configuration. After successfully running a test, the corpusDirectory will contain the coverage_report.html and other folders like call_sequences and tests_results.
q

## Running a coverage test with medusa
