import Highcharts from 'highcharts';

/**
 * Centralized Highcharts theme using RdMarket Intelligence design tokens.
 * Adheres strictly to Section 3.5 of the design specification.
 */
export function applyHighchartsTheme(isDark = true) {
  const textColor = isDark ? '#E6EAED' : '#1A1D20';
  const textMuted = isDark ? '#667180' : '#8B95A1';
  const textSecondary = isDark ? '#9AA5B1' : '#5A6472';
  const borderColor = isDark ? '#232A32' : '#DDDAD3';
  const bgRaised = isDark ? '#181D23' : '#EDECE8';
  const series1 = isDark ? '#E6EAED' : '#1A1D20'; // Actual rate
  const series2 = isDark ? '#D9A441' : '#B88228'; // SMA 7
  const series3 = isDark ? '#4FA3D9' : '#2A7FB8'; // SMA 30
  const series4 = isDark ? '#B48EE0' : '#8E62C4'; // SMA 90

  Highcharts.setOptions({
    colors: [series1, series2, series3, series4],
    chart: {
      backgroundColor: 'transparent',
      borderWidth: 0,
      borderRadius: 0,
      marginRight: 65,
      style: {
        fontFamily: '"IBM Plex Sans", -apple-system, BlinkMacSystemFont, sans-serif',
      },
      animation: false,
    },
    title: {
      text: undefined, // Panel header provides title
    },
    subtitle: {
      text: undefined,
    },
    credits: {
      enabled: false,
    },
    xAxis: {
      type: 'datetime',
      gridLineWidth: 0,
      lineWidth: 1,
      lineColor: borderColor,
      tickColor: borderColor,
      tickWidth: 1,
      labels: {
        style: {
          color: textMuted,
          fontFamily: '"JetBrains Mono", "IBM Plex Mono", monospace',
          fontSize: '11px',
        },
      },
      crosshair: {
        width: 1,
        color: borderColor,
        dashStyle: 'Dash',
        label: {
          enabled: true,
          backgroundColor: bgRaised,
          borderColor: borderColor,
          borderWidth: 1,
          borderRadius: 2,
          padding: 4,
          style: {
            color: textColor,
            fontFamily: '"JetBrains Mono", monospace',
            fontSize: '11px',
          },
        },
      },
    },
    yAxis: {
      opposite: true, // Financial convention: right-side axis
      gridLineWidth: 1,
      gridLineColor: borderColor,
      gridLineDashStyle: 'Solid',
      lineWidth: 1,
      lineColor: borderColor,
      title: {
        text: undefined,
      },
      labels: {
        align: 'left',
        x: 8,
        style: {
          color: textMuted,
          fontFamily: '"JetBrains Mono", "IBM Plex Mono", monospace',
          fontSize: '11px',
          fontWeight: '400',
        },
        formatter: function () {
          return new Intl.NumberFormat('id-ID', {
            maximumFractionDigits: 0,
          }).format(this.value as number);
        },
      },
      crosshair: {
        width: 1,
        color: borderColor,
        dashStyle: 'Dash',
        label: {
          enabled: true,
          backgroundColor: bgRaised,
          borderColor: borderColor,
          borderWidth: 1,
          borderRadius: 2,
          padding: 4,
          style: {
            color: textColor,
            fontFamily: '"JetBrains Mono", monospace',
            fontSize: '11px',
          },
          formatter: function (value: number) {
            return new Intl.NumberFormat('id-ID', {
              minimumFractionDigits: 2,
              maximumFractionDigits: 2,
            }).format(value);
          },
        },
      },
    },
    tooltip: {
      shared: true,
      useHTML: true,
      backgroundColor: bgRaised,
      borderColor: borderColor,
      borderWidth: 1,
      borderRadius: 2,
      shadow: false,
      padding: 8,
      style: {
        color: textColor,
        fontFamily: '"JetBrains Mono", monospace',
        fontSize: '11px',
      },
      xDateFormat: '%d %b %Y, %H:%M WIB',
      headerFormat: '<div style="color:' + textMuted + ';font-size:10px;text-transform:uppercase;margin-bottom:4px;letter-spacing:0.04em;">{point.key}</div>',
      pointFormatter: function () {
        const val = this.y !== null && this.y !== undefined
          ? new Intl.NumberFormat('id-ID', { minimumFractionDigits: 2, maximumFractionDigits: 2 }).format(this.y)
          : '—';
        return `<div style="display:flex;justify-content:space-between;gap:12px;margin:2px 0;">
          <span style="color:${this.color};font-weight:500;">${this.series.name}:</span>
          <span style="font-weight:600;font-variant-numeric:tabular-nums;">Rp ${val}</span>
        </div>`;
      },
    },
    legend: {
      enabled: false, // Customized in panel controls for compact density
    },
    plotOptions: {
      series: {
        animation: false,
        connectNulls: false, // Render real gaps for missing data
        states: {
          hover: {
            lineWidthPlus: 0,
          },
        },
        marker: {
          enabled: false,
          radius: 3,
          states: {
            hover: {
              enabled: true,
              radius: 4,
            },
          },
        },
      },
      line: {
        lineWidth: 1.5,
      },
    },
  });
}
