<?php

use Ruudk\DeadCodeAnalyzer\DeadCodeAnalyzer;

$path = __DIR__ . '/../../../../.dca.config.php';

if (true === file_exists($path)) {
    $config = require $path;
} else {
    $config = require __DIR__ . '/../.dca.config.php';
}

if (false === $config['enabled']) {
    return;
}

$analyzer = new DeadCodeAnalyzer(
    $config['allowedNamespaces'],
    $config['host'],
    $config['port'],
    $config['packetSize']
);
$analyzer->register();
