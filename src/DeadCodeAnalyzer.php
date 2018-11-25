<?php

declare(strict_types=1);

namespace Ruudk\DeadCodeAnalyzer;

use function count;
use function strlen;
use function strpos;

final class DeadCodeAnalyzer
{
    /**
     * @var int
     */
    private $packetSize;

    /**
     * @var array
     */
    private $metrics = [];

    /**
     * @var int
     */
    private $metricsLength = 0;

    /**
     * @var array
     */
    private $alowedNamespaces;

    /**
     * @var string
     */
    private $host;

    /**
     * @var int
     */
    private $port;

    public function __construct(array $alowedNamespaces = [], string $host = '127.0.0.1', int $port = 8125, int $packetSize = 500)
    {
        $this->alowedNamespaces = $alowedNamespaces;
        $this->host       = $host;
        $this->port       = $port;
        $this->packetSize = $packetSize;
    }

    public function loaded(string $class) : void
    {
        if (false === $this->isAllowed($class)) {
            return;
        }

        $metric = sprintf(
            'autoloaded,class=%s',
            str_replace("\\", '/', $class)
        );

        $this->metrics[]     = $metric;
        $this->metricsLength += strlen($metric);

        $this->flushIfNeeded();
    }

    private function isAllowed(string $class) : bool
    {
        if (0 === count($this->alowedNamespaces)) {
            return true;
        }

        foreach ($this->alowedNamespaces as $allowedNamespace) {
            if (false !== strpos($class, $allowedNamespace)) {
                return true;
            }
        }

        return false;
    }

    private function flushIfNeeded() : void
    {
        if ($this->metricsLength < $this->packetSize) {
            return;
        }

        $this->flush();
    }

    public function flush() : void
    {
        if (empty($this->metrics)) {
            return;
        }

        $totalLength = 0;
        $buffer      = [];
        foreach ($this->metrics as $metric) {
            $length = strlen($metric);
            if ($totalLength + $length < $this->packetSize) {
                $totalLength += $length;
                $buffer[]    = $metric;

                continue;
            }

            $this->increment($buffer);

            $totalLength = $length;
            $buffer      = [$metric];
        }

        if ($totalLength > 0) {
            $this->increment($buffer);
        }

        $this->metrics       = [];
        $this->metricsLength = 0;
    }

    private function increment(array $metrics, int $delta = 1) : bool
    {
        return $this->send(
            array_map(
                function ($metric) use ($delta) {
                    return $metric . ':' . $delta . '|c';
                },
                $metrics
            )
        );
    }

    private function send(array $data) : bool
    {
        $socket = @fsockopen('udp://' . $this->host, $this->port);
        if (false === $socket) {
            return false;
        }

        @fwrite($socket, implode("\n", $data) . "\n");

        fclose($socket);

        return true;
    }

    public function register() : void
    {
        spl_autoload_register([$this, 'loaded'], false, true);
        register_shutdown_function([$this, 'flush']);
    }
}

