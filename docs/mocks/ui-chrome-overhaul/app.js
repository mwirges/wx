(() => {
  const DIRECTIONS = {
    'snw-glass': {
      badge: 'Direction A · Recommended',
      summary: 'Enterprise Liquid Glass — Strange New Worlds cyan/gold telemetry fused with 24pt continuous squircles, frosted multi-layer glass, 1px specular rim refraction, swept horizon light-rails, and iOS 26 capsule+orb dock. Zero 90° brackets.'
    },
    'atmos-glass': {
      badge: 'Direction B · Weather-First',
      summary: 'Atmospheric Horizon — Ultra-translucent 28pt frosted glass floating over rich sky/storm gradients. Minimizes console framing while preserving crisp Starfleet telemetry callsigns and glowing status pips.'
    },
    'lcars-curved': {
      badge: 'Direction C · High Trek DNA',
      summary: 'Curved LCARS 2401 — Swept Starfleet L-elbow shoulders with C1-continuous concave inner fillets, segmented gold/periwinkle/cyan pill caps, and smoked sapphire-quartz glass. Zero 90° corners.'
    },
    'legacy-v1': {
      badge: 'Audit Baseline · Current v1',
      summary: 'Current v1 Chrome (Before) — 93% opaque flat navy cards, 5–8pt boxy radii, 52+ spiky 90° SNWCornerBrackets, cramped 6-tab bar, inverted iPhone Surface list, 240px static radar box, and Mac 115pt header + SYS.01 truncation.'
    }
  };

  const STATIONS = {
    newhaven: {
      place: 'New Haven, IN',
      short: 'New Haven',
      icao: 'KFWA',
      deltaF: 0,
      radarStation: 'NEW HAVEN ★',
      radarNeighbor: 'FORT WAYNE',
      radarSub1: 'HUNTINGTON',
      radarSub2: 'AUBURN'
    },
    milwaukee: {
      place: 'Milwaukee, WI',
      short: 'Milwaukee',
      icao: 'KMKE',
      deltaF: -6,
      radarStation: 'MILWAUKEE ★',
      radarNeighbor: 'WAUKESHA',
      radarSub1: 'RACINE',
      radarSub2: 'PORT WASH'
    },
    amarillo: {
      place: 'Amarillo, TX',
      short: 'Amarillo',
      icao: 'KAMA',
      deltaF: 9,
      radarStation: 'AMARILLO ★',
      radarNeighbor: 'CANYON',
      radarSub1: 'HEREFORD',
      radarSub2: 'PAMPA'
    }
  };

  const WEATHER_PRESETS = {
    rain: {
      obsTime: '6:45 PM',
      phoneClock: '6:45',
      tempF: 72,
      highF: 76,
      lowF: 58,
      feelsF: 71,
      dewF: 64,
      rhPct: 78,
      windMph: 9,
      gustMph: 17,
      windDir: 'SSW',
      windDeg: 205,
      pressInHg: '29.92',
      pressHpa: '1013 hPa',
      pressDesc: 'Falling slow',
      pressTrend: '↘',
      aqi: 28,
      pm25: '6.8',
      uv: '1.2',
      uvSub: 'Low · Cloud shielded',
      visMi: 6.5,
      visSub: 'Light mist ceiling',
      cond: 'Light Rain & Mist',
      icon: '#wx-icon-rain-sun',
      compactIcon: '#ico-wx-rain',
      islandIcon: '#ico-droplet',
      islandLead: '18m',
      islandBars: [
        [85, 'heavy'], [100, 'heavy'], [70, ''], [45, ''], [28, 'dry']
      ],
      nowcastTitle: 'Rain tapering off in ~18 min',
      nowcastSub: 'Light stratiform band moving ENE at 28 mph · 0.18" expected',
      qpfIn: 0.18,
      radarModePill: 'MRMS · Stratiform',
      radarPipColor: 'var(--wx-green)',
      statusPipColor: 'var(--wx-amber)',
      solar: {
        tag: '+15.4° WNW',
        aria: 'Solar horizon curve showing sun at +15.4 degrees behind evening rain clouds, sunset at 8:04 PM',
        progressD: 'M48 52 C85 24, 122 14, 155 14 C184 14, 204 26, 214 36',
        stroke: '#f5a623',
        orbX: 214,
        orbY: 36,
        orbFill: '#ffe082',
        haloFill: 'rgba(245,166,35,0.24)',
        ephemLeft: 'Waning Gibbous · 87%',
        ephemRight: '12h 52m Daylight'
      },
      sounding: {
        cape: '420',
        capeSub: 'J/kg · Moist layer',
        shearKt: 28,
        shearSub: 'Stratiform flow',
        srh: '95',
        srhSub: 'm²/s² · Modest',
        pwatIn: 1.42,
        pwatSub: '82nd percentile'
      },
      alert: {
        band: 'yellow',
        badge: 'COND · YELLOW',
        title: 'Areal Flood Watch',
        sub: 'Maumee River Basin · Until 9:00 PM EDT'
      },
      severeScreen: {
        panelHref: '#snw-condition-panel-yellow',
        cardBg: 'linear-gradient(155deg,rgba(88,54,12,0.56),rgba(18,20,38,0.52))',
        railColor: 'var(--wx-gold)',
        tag: 'CONDITION · YELLOW',
        tagColor: '#ffe9b8',
        tagBorder: 'rgba(245,166,35,0.55)',
        tagBg: 'rgba(245,166,35,0.22)',
        metaColor: '#ffe7b3',
        title: 'Areal Flood Watch',
        meta: 'Issued by NWS Northern Indiana · Until 9:00 PM EDT',
        body: (u) => u === 'metric'
          ? 'Stratiform rain band producing 5–11 mm additional runoff across low-lying Maumee Basin creeks moving ENE at 45 km/h.'
          : 'Stratiform rain band producing 0.18"–0.45" additional runoff across low-lying Maumee Basin creeks moving ENE at 28 mph.',
        spcTag: 'TSTM · GEN',
        spcHeadline: 'General Non-Severe Thunder Risk',
        spcSummary: 'Moist stratiform overrunning with embedded elevated rumbles; organized severe convection unlikely.',
        tor: '<2%',
        wind: '5%',
        hail: '<2%'
      },
      chase: {
        ceiling: 'CEILING · TSTM',
        day1: 'Day 1 · TSTM (Gen)',
        day2: 'Day 2 · MRGL (1/5)',
        day3: 'Day 3 · NONE',
        mcdTitle: 'HYDROLOGY MESOSCALE NOTE · MAUMEE BASIN',
        mcdProb: 'WATCH PROB 15%',
        mcdDesc: 'Warm-conveyor stratiform rain band training ENE; localized ponding possible in poor-drainage areas.',
        clusterKicker: 'Cluster #1 · Maumee Basin Hydrologic Watch',
        scoreTag: 'SCORE · 34',
        clusterHeadline: '1 Active Flood Watch · Nearest NEXRAD KIWX'
      },
      bars: [
        [88, 'heavy'], [95, 'heavy'], [76, ''], [64, ''],
        [48, ''], [34, ''], [22, ''], [14, 'dry'],
        [14, 'dry'], [14, 'dry'], [14, 'dry'], [26, ''],
        [38, ''], [24, ''], [14, 'dry'], [14, 'dry']
      ],
      hourly: [
        { time: 'Now', icon: '#ico-wx-rain', pop: '70%', dF: 0 },
        { time: '7 PM', icon: '#ico-wx-sun-rain', pop: '45%', dF: -2 },
        { time: '8 PM', icon: '#ico-wx-sunset', pop: 'Set', gold: true, dF: -5 },
        { time: '9 PM', icon: '#ico-wx-cloud', pop: '15%', dF: -8 },
        { time: '10 PM', icon: '#ico-wx-moon', pop: '', dF: -11 }
      ],
      todayRow: { name: 'Today', icon: '#ico-wx-sun-rain', pop: '70%' },
      frameTimes: ['6:20 PM', '6:25 PM', '6:30 PM', '6:35 PM', '6:40 PM', '6:45 PM']
    },
    severe: {
      obsTime: '6:45 PM',
      phoneClock: '6:45',
      tempF: 78,
      highF: 84,
      lowF: 63,
      feelsF: 82,
      dewF: 71,
      rhPct: 82,
      windMph: 22,
      gustMph: 44,
      windDir: 'WSW',
      windDeg: 245,
      pressInHg: '29.68',
      pressHpa: '1005 hPa',
      pressDesc: 'Falling rapid',
      pressTrend: '↓↓',
      aqi: 34,
      pm25: '8.1',
      uv: '0.6',
      uvSub: 'Low · Dense anvil',
      visMi: 3.5,
      visSub: 'Heavy rain & hail',
      cond: 'Severe Thunderstorms',
      icon: '#wx-icon-severe-storm',
      compactIcon: '#ico-wx-tstorm',
      islandIcon: '#ico-alert',
      islandLead: '9m',
      islandBars: [
        [55, ''], [92, 'heavy'], [100, 'heavy'], [88, 'heavy'], [64, '']
      ],
      nowcastTitle: 'Heavy core arriving in 9 min',
      nowcastSub: 'Discrete supercell WSW · Peak rate 1.45"/hr + 1.00" hail',
      qpfIn: 1.12,
      radarModePill: 'MESO · 66 dBZ Core',
      radarPipColor: 'var(--wx-red)',
      statusPipColor: 'var(--wx-red)',
      solar: {
        tag: '+15.4° WNW',
        aria: 'Solar horizon curve showing sun shadowed behind severe thunderstorm anvil at +15.4 degrees',
        progressD: 'M48 52 C85 24, 122 14, 155 14 C184 14, 204 26, 214 36',
        stroke: '#ff9f0a',
        orbX: 214,
        orbY: 36,
        orbFill: '#ffb347',
        haloFill: 'rgba(255,59,86,0.28)',
        ephemLeft: 'Waning Gibbous · 87%',
        ephemRight: 'Anvil Shadowed'
      },
      sounding: {
        cape: '3,250',
        capeSub: 'J/kg · Extreme',
        shearKt: 54,
        shearSub: 'Supercell support',
        srh: '310',
        srhSub: 'm²/s² · Strong veer',
        pwatIn: 1.88,
        pwatSub: '97th percentile'
      },
      alert: {
        band: 'red',
        badge: 'COND · RED',
        title: 'Severe Thunderstorm Warning',
        sub: '60 mph gusts & quarter hail · Until 7:30 PM EDT'
      },
      severeScreen: {
        panelHref: '#snw-condition-panel',
        cardBg: 'linear-gradient(155deg,rgba(90,18,36,0.58),rgba(20,14,34,0.52))',
        railColor: 'var(--wx-red)',
        tag: 'CONDITION · RED',
        tagColor: '#ffb4c0',
        tagBorder: 'rgba(255,92,116,0.55)',
        tagBg: 'rgba(255,59,86,0.22)',
        metaColor: '#ffd8df',
        title: 'Severe Thunderstorm Warning',
        meta: 'Issued by NWS Northern Indiana · Expires 7:30 PM EDT',
        body: (u) => u === 'metric'
          ? '97 km/h wind gusts and quarter-size hail (25 mm) radar indicated along a line from Roanoke to New Haven moving NE at 56 km/h.'
          : '60 mph wind gusts and quarter-size hail (1.00") radar indicated along a line from Roanoke to New Haven moving NE at 35 mph.',
        spcTag: 'SLGT · 2/5',
        spcHeadline: 'Scattered Severe Storms Possible',
        spcSummary: 'Discrete warm-sector supercells across northern Indiana transitioning into bowing segments this evening.',
        tor: '5%',
        wind: '15%',
        hail: '15%'
      },
      chase: {
        ceiling: 'CEILING · ENH',
        day1: 'Day 1 · SLGT (2/5)',
        day2: 'Day 2 · ENH (3/5)',
        day3: 'Day 3 · MRGL (1/5)',
        mcdTitle: 'MCD #0842 · NORTHERN INDIANA / NW OHIO',
        mcdProb: 'WATCH PROB 60%',
        mcdDesc: 'Steep mid-level lapse rates & 45–50 kt effective bulk shear supporting supercell organization.',
        clusterKicker: 'Cluster #1 · Wabash–Maumee Corridor',
        scoreTag: 'SCORE · 118',
        clusterHeadline: '4 Active Severe Cells · Nearest NEXRAD KIWX'
      },
      bars: [
        [38, ''], [64, ''], [96, 'heavy'], [100, 'heavy'],
        [94, 'heavy'], [88, 'heavy'], [78, 'heavy'], [62, ''],
        [48, ''], [36, ''], [26, ''], [18, 'dry'],
        [16, 'dry'], [16, 'dry'], [16, 'dry'], [16, 'dry']
      ],
      hourly: [
        { time: 'Now', icon: '#ico-wx-tstorm', pop: '95%', dF: 0 },
        { time: '7 PM', icon: '#ico-wx-tstorm', pop: '85%', dF: -4 },
        { time: '8 PM', icon: '#ico-wx-rain', pop: '55%', dF: -9 },
        { time: '9 PM', icon: '#ico-wx-sun-rain', pop: '25%', dF: -11 },
        { time: '10 PM', icon: '#ico-wx-cloud', pop: '10%', dF: -13 }
      ],
      todayRow: { name: 'Today', icon: '#ico-wx-tstorm', pop: '95%' },
      frameTimes: ['6:20 PM', '6:25 PM', '6:30 PM', '6:35 PM', '6:40 PM', '6:45 PM']
    },
    golden: {
      obsTime: '7:42 PM',
      phoneClock: '7:42',
      tempF: 68,
      highF: 71,
      lowF: 49,
      feelsF: 68,
      dewF: 49,
      rhPct: 52,
      windMph: 6,
      gustMph: 11,
      windDir: 'WNW',
      windDeg: 290,
      pressInHg: '30.18',
      pressHpa: '1022 hPa',
      pressDesc: 'Rising steady',
      pressTrend: '↗',
      aqi: 31,
      pm25: '7.4',
      uv: '0.5',
      uvSub: 'Low · Setting sun',
      visMi: 10.0,
      visSub: 'Clear golden horizon',
      cond: 'Clear · Golden Hour',
      icon: '#wx-icon-golden-clear',
      compactIcon: '#ico-wx-sun',
      islandIcon: '#ico-tab-surface',
      islandLead: 'Clear',
      islandBars: [
        [24, 'dry'], [24, 'dry'], [24, 'dry'], [24, 'dry'], [24, 'dry']
      ],
      nowcastTitle: 'All Clear · Dry next 6 hours',
      nowcastSub: 'High pressure ridge overhead · Visibility 10 mi',
      qpfIn: 0,
      radarModePill: 'Clear Air · 8 dBZ',
      radarPipColor: 'var(--wx-cyan)',
      statusPipColor: 'var(--wx-green)',
      solar: {
        tag: '+4.6° WNW',
        aria: 'Solar horizon curve showing golden-hour sun low at +4.6 degrees just above 8:04 PM sunset horizon',
        progressD: 'M48 52 C85 24, 122 14, 155 14 C196 14, 224 34, 240 48',
        stroke: '#f5a623',
        orbX: 240,
        orbY: 48,
        orbFill: '#ffe082',
        haloFill: 'rgba(245,166,35,0.44)',
        ephemLeft: 'Golden Hour · Set 8:04P',
        ephemRight: '12h 52m Daylight'
      },
      sounding: {
        cape: '80',
        capeSub: 'J/kg · Capped ridge',
        shearKt: 18,
        shearSub: 'Fair-weather breeze',
        srh: '45',
        srhSub: 'm²/s² · Weak',
        pwatIn: 0.74,
        pwatSub: 'Dry airmass'
      },
      alert: null,
      severeScreen: {
        panelHref: '#snw-condition-panel-nominal',
        cardBg: 'linear-gradient(155deg,rgba(14,48,42,0.46),rgba(14,22,40,0.48))',
        railColor: 'var(--wx-green)',
        tag: 'ALL CLEAR · NOMINAL',
        tagColor: '#b9f6ca',
        tagBorder: 'rgba(48,209,88,0.48)',
        tagBg: 'rgba(48,209,88,0.16)',
        metaColor: '#b9f6ca',
        title: 'No Active NWS Watches or Warnings',
        meta: 'Surface ridge overhead · Next 24h hazard-free',
        body: (u) => u === 'metric'
          ? 'Subsident high-pressure ridge keeping northern Indiana dry with 16 km visibility and light WNW breeze.'
          : 'Subsident high-pressure ridge keeping northern Indiana dry with 10 mi visibility and light WNW breeze.',
        spcTag: 'NONE · QUIET',
        spcHeadline: 'No Severe Thunderstorms Forecast',
        spcSummary: 'Strong mid-level capping inversion and dry column (PWAT < 20 mm) suppressing deep convection.',
        tor: '0%',
        wind: '0%',
        hail: '0%'
      },
      chase: {
        ceiling: 'CEILING · QUIET',
        day1: 'Day 1 · NONE',
        day2: 'Day 2 · TSTM (West)',
        day3: 'Day 3 · MRGL (Plains)',
        mcdTitle: 'SYNOPTIC BRIEF · GREAT LAKES SURFACE RIDGE',
        mcdProb: 'WATCH PROB 0%',
        mcdDesc: 'Broad anticyclonic subsidence overhead; zero mesoscale discussions or convective watches in effect.',
        clusterKicker: 'CONUS Sector · Quiet Fair-Weather Pattern',
        scoreTag: 'SCORE · 0',
        clusterHeadline: '0 Active Severe Cells · Nearest NEXRAD KIWX'
      },
      bars: Array.from({ length: 16 }, () => [14, 'dry']),
      hourly: [
        { time: 'Now', icon: '#ico-wx-sun', pop: '', dF: 0 },
        { time: '8 PM', icon: '#ico-wx-sunset', pop: 'Set', gold: true, dF: -3 },
        { time: '9 PM', icon: '#ico-wx-moon', pop: '', dF: -7 },
        { time: '10 PM', icon: '#ico-wx-moon', pop: '', dF: -10 },
        { time: '11 PM', icon: '#ico-wx-moon', pop: '', dF: -12 }
      ],
      todayRow: { name: 'Today', icon: '#ico-wx-sun', pop: '' },
      frameTimes: ['7:17 PM', '7:22 PM', '7:27 PM', '7:32 PM', '7:37 PM', '7:42 PM']
    },
    night: {
      obsTime: '10:15 PM',
      phoneClock: '10:15',
      tempF: 44,
      highF: 64,
      lowF: 41,
      feelsF: 41,
      dewF: 39,
      rhPct: 84,
      windMph: 5,
      gustMph: 9,
      windDir: 'SSW',
      windDeg: 200,
      pressInHg: '30.24',
      pressHpa: '1024 hPa',
      pressDesc: 'Steady high',
      pressTrend: '→',
      aqi: 24,
      pm25: '5.6',
      uv: '0',
      uvSub: 'Night · Below horizon',
      visMi: 10.0,
      visSub: 'Crisp starlight',
      cond: 'Mostly Clear · Starlight',
      icon: '#wx-icon-orbital-night',
      compactIcon: '#ico-wx-moon',
      islandIcon: '#ico-wx-moon',
      islandLead: 'Clear',
      islandBars: [
        [24, 'dry'], [24, 'dry'], [24, 'dry'], [24, 'dry'], [24, 'dry']
      ],
      nowcastTitle: 'Dry overnight · Crisp starlight',
      nowcastSub: 'Radiational cooling under calm breeze · Waning Gibbous 87%',
      qpfIn: 0,
      radarModePill: 'Clear Air · 6 dBZ',
      radarPipColor: 'var(--wx-periwinkle)',
      statusPipColor: 'var(--wx-green)',
      solar: {
        tag: '-28.4° NNW',
        aria: 'Nocturnal solar curve showing sun 28.4 degrees below horizon with waning gibbous moonlight',
        progressD: 'M246 52 C260 58, 274 62, 286 63',
        stroke: '#7cecff',
        orbX: 286,
        orbY: 63,
        orbFill: '#7cecff',
        haloFill: 'rgba(94,114,255,0.36)',
        ephemLeft: 'Waning Gibbous · 87%',
        ephemRight: 'Rise Tomorrow 7:13A'
      },
      sounding: {
        cape: '0',
        capeSub: 'J/kg · Nocturnal cap',
        shearKt: 16,
        shearSub: 'Calm boundary',
        srh: '60',
        srhSub: 'm²/s² · Nocturnal LLJ',
        pwatIn: 0.62,
        pwatSub: 'Crisp dry column'
      },
      alert: null,
      severeScreen: {
        panelHref: '#snw-condition-panel-nominal',
        cardBg: 'linear-gradient(155deg,rgba(14,38,52,0.48),rgba(10,16,34,0.52))',
        railColor: 'var(--wx-green)',
        tag: 'ALL CLEAR · NOMINAL',
        tagColor: '#b9f6ca',
        tagBorder: 'rgba(48,209,88,0.48)',
        tagBg: 'rgba(48,209,88,0.16)',
        metaColor: '#b9f6ca',
        title: 'No Active Nocturnal Hazards',
        meta: 'Calm radiational cooling · Next 12h hazard-free',
        body: (u) => u === 'metric'
          ? 'Clear starlit boundary layer with 16 km visibility and calm 8 km/h SSW nocturnal flow.'
          : 'Clear starlit boundary layer with 10 mi visibility and calm 5 mph SSW nocturnal flow.',
        spcTag: 'NONE · QUIET',
        spcHeadline: 'No Nocturnal Convection Expected',
        spcSummary: 'Zero surface-based CAPE beneath nocturnal inversion; tranquil conditions through daybreak.',
        tor: '0%',
        wind: '0%',
        hail: '0%'
      },
      chase: {
        ceiling: 'CEILING · QUIET',
        day1: 'Day 1 · NONE',
        day2: 'Day 2 · TSTM (Plains)',
        day3: 'Day 3 · MRGL (1/5)',
        mcdTitle: 'NOCTURNAL BOUNDARY LAYER · ALL CLEAR',
        mcdProb: 'WATCH PROB 0%',
        mcdDesc: 'Radiational cooling decoupling surface winds; zero convective watches or warnings nationwide.',
        clusterKicker: 'CONUS Sector · Nocturnal Quiet Regime',
        scoreTag: 'SCORE · 0',
        clusterHeadline: '0 Active Severe Cells · Nearest NEXRAD KIWX'
      },
      bars: Array.from({ length: 16 }, () => [14, 'dry']),
      hourly: [
        { time: 'Now', icon: '#ico-wx-moon', pop: '', dF: 0 },
        { time: '11 PM', icon: '#ico-wx-moon', pop: '', dF: -1 },
        { time: '12 AM', icon: '#ico-wx-moon', pop: '', dF: -2 },
        { time: '1 AM', icon: '#ico-wx-fog', pop: '10%', dF: -3 },
        { time: '2 AM', icon: '#ico-wx-moon', pop: '', dF: -3 }
      ],
      todayRow: { name: 'Tonight', icon: '#ico-wx-moon', pop: '' },
      frameTimes: ['9:50 PM', '9:55 PM', '10:00 PM', '10:05 PM', '10:10 PM', '10:15 PM']
    }
  };

  const RADAR_OFFSETS = [
    { age: '-25m', shiftX: -20, shiftY: 11, scale: 0.94 },
    { age: '-20m', shiftX: -16, shiftY: 8, scale: 0.96 },
    { age: '-15m', shiftX: -11, shiftY: 6, scale: 0.98 },
    { age: '-10m', shiftX: -7, shiftY: 4, scale: 0.99 },
    { age: '-5m', shiftX: -3, shiftY: 2, scale: 1.0 },
    { age: 'LIVE', shiftX: 0, shiftY: 0, scale: 1.02 }
  ];

  const MAC_MODES = ['Tactical', 'Surface', 'Radar', 'Outlooks', 'Chase', 'Climate', 'Tropics', 'Grid'];

  const root = document.documentElement;
  let currentWeatherKey = 'rain';
  let currentStationKey = 'newhaven';
  let customPlaceOverride = null;
  let currentUnits = 'imperial';
  let currentMacMode = 'Tactical';
  let radarTimer = null;
  let radarPlaying = false;
  let currentFrame = 5;

  function fmtTemp(f) {
    const val = currentUnits === 'metric' ? Math.round((f - 32) * 5 / 9) : Math.round(f);
    return `${val}°`;
  }

  function fmtSpeed(mph) {
    return currentUnits === 'metric'
      ? `${Math.round(mph * 1.60934)} km/h`
      : `${Math.round(mph)} mph`;
  }

  function fmtVis(mi) {
    if (currentUnits === 'metric') {
      const km = mi * 1.60934;
      return `${km >= 15 ? Math.round(km) : km.toFixed(1)} km`;
    }
    return `${Number.isInteger(mi) ? mi : mi.toFixed(1)} mi`;
  }

  function withTransition(fn, focusTarget, immediate = false) {
    const reduced = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
    if (!immediate && !reduced && root.dataset.ready === 'true' && document.startViewTransition) {
      const vt = document.startViewTransition(() => fn());
      if (focusTarget) {
        vt.finished.finally(() => focusTarget.focus({ preventScroll: true }));
      }
    } else {
      fn();
      if (focusTarget) {
        focusTarget.focus({ preventScroll: true });
      }
    }
  }

  // 1. Studio Top-Level Stage Tabs (ARIA tablist + hidden="until-found" + beforematch + scroll-fade fallback)
  let selectStudioTabById = null;
  function initStudioTabs() {
    const tablist = document.querySelector('[data-studio-tablist]');
    if (!tablist) return;
    const tabs = Array.from(tablist.querySelectorAll('[role="tab"]'));
    const panels = tabs.map((t) => document.getElementById(t.dataset.panel));

    function selectStudioTab(nextTab, moveFocus = false, immediate = false) {
      withTransition(() => {
        tabs.forEach((tab, idx) => {
          const active = tab === nextTab;
          tab.setAttribute('aria-selected', active ? 'true' : 'false');
          tab.tabIndex = active ? 0 : -1;
          const panel = panels[idx];
          if (!panel) return;
          if (active) {
            panel.removeAttribute('hidden');
          } else {
            panel.setAttribute('hidden', 'until-found');
          }
        });
      }, moveFocus ? nextTab : null, immediate);

      if (!immediate) {
        nextTab.scrollIntoView({ block: 'nearest', inline: 'nearest', container: 'nearest' });
      }
    }

    selectStudioTabById = (id, immediate = false) => {
      const target = document.getElementById(`tab-${id}`);
      if (target) selectStudioTab(target, false, immediate);
    };

    tabs.forEach((tab, idx) => {
      tab.addEventListener('click', () => selectStudioTab(tab, false));
      tab.addEventListener('keydown', (e) => {
        let nextIdx = null;
        if (e.key === 'ArrowRight') nextIdx = (idx + 1) % tabs.length;
        else if (e.key === 'ArrowLeft') nextIdx = (idx - 1 + tabs.length) % tabs.length;
        else if (e.key === 'Home') nextIdx = 0;
        else if (e.key === 'End') nextIdx = tabs.length - 1;
        if (nextIdx !== null) {
          e.preventDefault();
          selectStudioTab(tabs[nextIdx], true);
        }
      });

      const panel = panels[idx];
      if (panel) {
        panel.addEventListener('beforematch', () => selectStudioTab(tab, false));
      }
    });

    if (window.CSS && !CSS.supports('animation-timeline', 'scroll()') && 'IntersectionObserver' in window) {
      const first = tablist.firstElementChild;
      const last = tablist.lastElementChild;
      if (first && last) {
        const obs = new IntersectionObserver((entries) => {
          for (const { target, intersectionRatio } of entries) {
            tablist.classList.toggle(target === first ? 'more-start' : 'more-end', intersectionRatio < 1);
          }
        }, { root: tablist, threshold: 1, rootMargin: '1px' });
        obs.observe(first);
        obs.observe(last);
      }
    }
  }

  // 2. Theme Direction Switcher
  function setDirection(dir) {
    if (!DIRECTIONS[dir]) return;
    root.dataset.direction = dir;
    document.querySelectorAll('[data-set-direction]').forEach((btn) => {
      btn.setAttribute('aria-pressed', btn.dataset.setDirection === dir ? 'true' : 'false');
    });
    const badgeEl = document.getElementById('direction-badge');
    const summaryEl = document.getElementById('direction-summary');
    if (badgeEl) badgeEl.textContent = DIRECTIONS[dir].badge;
    if (summaryEl) summaryEl.textContent = DIRECTIONS[dir].summary;
  }

  // 3. Cohesive Weather + Station + Units + Ephemeris + Sounding + Radar Renderer
  function renderWeatherAndUnits() {
    const preset = WEATHER_PRESETS[currentWeatherKey] || WEATHER_PRESETS.rain;
    const station = customPlaceOverride || STATIONS[currentStationKey] || STATIONS.newhaven;
    const dF = station.deltaF || 0;

    const tempStr = fmtTemp(preset.tempF + dF);
    const highStr = fmtTemp(preset.highF + dF);
    const lowStr = fmtTemp(preset.lowF + dF);
    const feelsStr = fmtTemp(preset.feelsF + dF);
    const dewStr = fmtTemp(preset.dewF + dF);
    const windStr = fmtSpeed(preset.windMph);
    const gustVal = currentUnits === 'metric' ? Math.round(preset.gustMph * 1.60934) : preset.gustMph;
    const pwatStr = currentUnits === 'metric'
      ? `${(preset.sounding.pwatIn * 25.4).toFixed(0)} mm`
      : `${preset.sounding.pwatIn.toFixed(2)}"`;

    const normalHighF = 73 + dF;
    const normalLowF = 53 + dF;
    const departF = (preset.tempF + dF) - normalHighF;
    const departVal = currentUnits === 'metric' ? (departF * 5 / 9) : departF;
    const departSign = departVal > 0 ? '+' : '';
    const departTag = Math.abs(departF) <= 3 ? 'Nominal' : (departF > 0 ? 'Above normal' : 'Below normal');

    const sev = preset.severeScreen;
    const ch = preset.chase;
    const tropSust = fmtSpeed(115);
    const tropMove = fmtSpeed(16);
    const investMove = fmtSpeed(15);

    const bindings = {
      obsTime: preset.obsTime,
      phoneClock: preset.phoneClock,
      place: station.place,
      placeShort: station.short,
      placeUpper: station.place.toUpperCase(),
      icao: station.icao,
      temp: tempStr,
      cond: preset.cond,
      hl: `H: ${highStr}  L: ${lowStr}  ·  Feels ${feelsStr}`,
      hlCompact: `H:${highStr} L:${lowStr}`,
      legacyTodayHigh: highStr,
      legacyTonightLow: lowStr,
      feelsLabel: `Feels ${feelsStr}`,
      windChip: `↘ ${windStr} ${preset.windDir}`,
      windChipPlain: `${windStr} ${preset.windDir}`,
      windVal: windStr,
      windSub: `${preset.windDir} (${preset.windDeg}°) · Gust ${gustVal}`,
      rhVal: `${preset.rhPct}%`,
      rhChip: `${preset.rhPct}% RH`,
      dewVal: dewStr,
      dewSub: `Dew point ${dewStr} · ${preset.dewF + dF >= 68 ? 'Tropical' : preset.dewF + dF >= 60 ? 'Muggy' : 'Crisp'}`,
      pressTrend: preset.pressTrend,
      pressVal: currentUnits === 'metric' ? preset.pressHpa : preset.pressInHg,
      pressSub: currentUnits === 'metric'
        ? `${preset.pressInHg} inHg · ${preset.pressDesc}`
        : `${preset.pressHpa} · ${preset.pressDesc}`,
      aqiVal: String(preset.aqi),
      aqiCat: 'Good',
      aqiChip: `AQI ${preset.aqi}`,
      aqiSub: `PM2.5 ${preset.pm25} µg/m³`,
      pm25Val: `${preset.pm25} µg/m³`,
      uvVal: preset.uv,
      uvSub: preset.uvSub,
      visVal: fmtVis(preset.visMi),
      visSub: preset.visSub,
      solarTag: preset.solar.tag,
      ephemLeft: preset.solar.ephemLeft,
      ephemRight: preset.solar.ephemRight,
      capeVal: preset.sounding.cape,
      capeSub: preset.sounding.capeSub,
      shearVal: `${preset.sounding.shearKt} kt`,
      shearSub: preset.sounding.shearSub,
      srhVal: preset.sounding.srh,
      srhSub: preset.sounding.srhSub,
      pwatVal: pwatStr,
      pwatSub: preset.sounding.pwatSub,
      radarModePill: preset.radarModePill,
      unitBadge: currentUnits === 'metric' ? '°C · Metric' : '°F · Imperial',
      islandLead: preset.islandLead,
      nowcastTitle: preset.nowcastTitle,
      nowcastSub: preset.nowcastSub,
      qpfTotal: currentUnits === 'metric'
        ? `${(preset.qpfIn * 25.4).toFixed(1)} mm total`
        : `${preset.qpfIn.toFixed(2)}" total`,
      alertBadge: preset.alert ? preset.alert.badge : '',
      alertTitle: preset.alert ? preset.alert.title : '',
      alertSub: preset.alert
        ? (currentUnits === 'metric' ? preset.alert.sub.replace('60 mph', '97 km/h') : preset.alert.sub)
        : '',
      severeTag: sev.tag,
      severeTitle: sev.title,
      severeMeta: sev.meta,
      severeBody: sev.body(currentUnits),
      spcTag: sev.spcTag,
      spcHeadline: sev.spcHeadline,
      spcSummary: sev.spcSummary,
      spcTor: sev.tor,
      spcWind: sev.wind,
      spcHail: sev.hail,
      tropicsMeta: `${tropSust} sustained · 952 mb · NNE at ${tropMove}`,
      macTropicsSub: `24.6°N 84.9°W · Moving NNE at ${tropMove} · Hurricane-force winds extend outward 60 nm.`,
      macTropicsMax: `${tropSust} (100 kt)`,
      macInvestSub: `48-hour formation chance: 30% (Low) · 7-day formation chance: 60% (Medium) moving WNW at ${investMove}.`,
      chaseCeilingTag: ch.ceiling,
      chaseDay1: ch.day1,
      chaseDay2: ch.day2,
      chaseDay3: ch.day3,
      chaseMcdTitle: ch.mcdTitle,
      chaseMcdProb: ch.mcdProb,
      chaseMcdDesc: ch.mcdDesc,
      chaseClusterKicker: ch.clusterKicker,
      chaseScoreTag: ch.scoreTag,
      chaseClusterHeadline: ch.clusterHeadline,
      climNormal: `${fmtTemp(normalHighF)} / ${fmtTemp(normalLowF)}`,
      climDepart: `Departure: ${departSign}${departVal.toFixed(1)}° (${departTag})`,
      climRecHigh: `${fmtTemp(91 + dF)} (1953)`,
      climRecLow: `Record Low: ${fmtTemp(34 + dF)} (1974)`,
      climPrecip: currentUnits === 'metric' ? '46.7 mm' : '1.84 in',
      gridMetaNH: `HUM ${preset.rhPct}% · ${preset.windDir} ${fmtSpeed(preset.windMph)}`,
      gridMetaMKE: `HUM 64% · ENE ${fmtSpeed(11)}`,
      gridMetaAMA: `HUM 28% · SW ${fmtSpeed(18)}`
    };

    Object.entries(bindings).forEach(([key, val]) => {
      document.querySelectorAll(`[data-bind="${key}"]`).forEach((el) => {
        el.textContent = val;
      });
    });

    // Update Sky pills for unit consistency
    const skyLabels = [
      ['rain', `Rain ${fmtTemp(WEATHER_PRESETS.rain.tempF + dF)}`],
      ['severe', `Condition Red ${fmtTemp(WEATHER_PRESETS.severe.tempF + dF)}`],
      ['golden', `Golden Hr ${fmtTemp(WEATHER_PRESETS.golden.tempF + dF)}`],
      ['night', `Night ${fmtTemp(WEATHER_PRESETS.night.tempF + dF)}`]
    ];
    skyLabels.forEach(([key, label]) => {
      const btn = document.querySelector(`[data-set-weather="${key}"]`);
      if (btn) btn.textContent = label;
    });

    // Update hero + Dynamic Island SVG <use> references
    document.querySelectorAll('[data-hero-icon]').forEach((useEl) => {
      useEl.setAttribute('href', preset.icon);
    });
    document.querySelectorAll('[data-island-icon]').forEach((useEl) => {
      useEl.setAttribute('href', preset.islandIcon);
    });

    // Update radar & menu-bar status pips
    document.querySelectorAll('[data-radar-pip]').forEach((pip) => {
      pip.style.setProperty('--pip-color', preset.radarPipColor);
    });
    document.querySelectorAll('[data-status-pip]').forEach((pip) => {
      pip.style.setProperty('--pip-color', preset.statusPipColor);
    });

    // Update Solar & Lunar Orbit SVG curves across Tactical & Mac Surface views
    document.querySelectorAll('[data-solar-svg]').forEach((svg) => {
      svg.setAttribute('aria-label', preset.solar.aria);
      const prog = svg.querySelector('[data-solar-progress]');
      const orb = svg.querySelector('[data-solar-orb]');
      const halo = svg.querySelector('[data-solar-halo]');
      if (prog) {
        prog.setAttribute('d', preset.solar.progressD);
        prog.setAttribute('stroke', preset.solar.stroke);
      }
      if (orb) {
        orb.setAttribute('cx', String(preset.solar.orbX));
        orb.setAttribute('cy', String(preset.solar.orbY));
        orb.setAttribute('fill', preset.solar.orbFill);
      }
      if (halo) {
        halo.setAttribute('cx', String(preset.solar.orbX));
        halo.setAttribute('cy', String(preset.solar.orbY));
        halo.setAttribute('fill', preset.solar.haloFill);
      }
    });

    // Update conditional alert ribbons & Condition Yellow vs Condition Red styling
    document.querySelectorAll('[data-alert-conditional]').forEach((el) => {
      if (!preset.alert) {
        el.hidden = true;
      } else {
        el.hidden = false;
        if (el.classList.contains('wx-alert-ribbon')) {
          el.classList.toggle('cond-yellow', preset.alert.band === 'yellow');
        }
      }
    });

    // Update 16-bar Nowcast timelines (including Lock Screen rectangular widget!)
    document.querySelectorAll('[data-nowcast-bars]').forEach((container) => {
      const bars = container.children;
      preset.bars.forEach(([heightPct, variant], idx) => {
        const bar = bars[idx];
        if (!bar) return;
        bar.style.height = `${heightPct}%`;
        bar.className = variant ? `nowcast-bar ${variant}` : 'nowcast-bar';
      });
    });

    // Update 5-bar Dynamic Island capsule sparklines
    document.querySelectorAll('[data-island-bars]').forEach((container) => {
      const bars = container.children;
      preset.islandBars.forEach(([heightPct, variant], idx) => {
        const bar = bars[idx];
        if (!bar) return;
        bar.style.height = `${heightPct}%`;
        bar.className = variant ? `island-mini-bar ${variant}` : 'island-mini-bar';
      });
    });

    // Update Hourly Horizon strips with vector SVG <use> icons across all stages
    document.querySelectorAll('[data-hourly-strip]').forEach((strip) => {
      const pills = strip.querySelectorAll('.hour-pill');
      preset.hourly.forEach((slot, idx) => {
        const pill = pills[idx];
        if (!pill) return;
        const timeEl = pill.querySelector('.hour-time');
        const useEl = pill.querySelector('use');
        const popEl = pill.querySelector('.hour-pop');
        const tempEl = pill.querySelector('.hour-temp');
        if (timeEl) timeEl.textContent = slot.time;
        if (useEl) useEl.setAttribute('href', slot.icon);
        if (popEl) {
          popEl.textContent = slot.pop || '\u00A0';
          popEl.style.color = slot.gold ? 'var(--wx-gold)' : '';
        }
        if (tempEl) tempEl.textContent = fmtTemp(preset.tempF + dF + slot.dF);
      });
    });

    // Update Legacy v1 Hourly strip so Before mode stays coherent across Sky & Metric toggles
    document.querySelectorAll('[data-legacy-hourly]').forEach((strip) => {
      const times = strip.querySelectorAll('[data-lh-time]');
      const temps = strip.querySelectorAll('[data-lh-temp]');
      preset.hourly.forEach((slot, idx) => {
        if (times[idx]) times[idx].textContent = slot.time;
        if (temps[idx]) temps[idx].textContent = fmtTemp(preset.tempF + dF + slot.dF);
      });
    });

    // Update Interactive iPhone Severe ConditionPanel card & styling per weather regime
    document.querySelectorAll('[data-severe-panel-use]').forEach((u) => u.setAttribute('href', sev.panelHref));
    document.querySelectorAll('[data-severe-card]').forEach((c) => { c.style.background = sev.cardBg; });
    document.querySelectorAll('[data-severe-rail]').forEach((r) => r.style.setProperty('--rail-color', sev.railColor));
    document.querySelectorAll('[data-severe-tag]').forEach((t) => {
      t.style.color = sev.tagColor;
      t.style.borderColor = sev.tagBorder;
      t.style.background = sev.tagBg;
    });
    document.querySelectorAll('[data-severe-meta]').forEach((m) => { m.style.color = sev.metaColor; });

    // Update New Haven pinned station rows across Config, Mac Grid & Popover so they never contradict active Sky
    document.querySelectorAll('[data-newhaven-temp]').forEach((el) => {
      el.textContent = fmtTemp(preset.tempF);
    });
    document.querySelectorAll('[data-newhaven-icon]').forEach((el) => {
      el.setAttribute('href', preset.compactIcon);
    });

    // Update Radar map city & station labels when switching active station
    const radarSt = station.radarStation || `${station.short.toUpperCase()} ★`;
    const radarNb = station.radarNeighbor || 'REGIONAL';
    document.querySelectorAll('[data-radar-station]').forEach((el) => { el.textContent = radarSt; });
    document.querySelectorAll('[data-radar-station-slash]').forEach((el) => {
      el.textContent = `${station.short.toUpperCase()} / ${station.icao} ★`;
    });
    document.querySelectorAll('[data-radar-neighbor]').forEach((el) => { el.textContent = radarNb; });
    document.querySelectorAll('[data-radar-sub1]').forEach((el) => { el.textContent = station.radarSub1 || ''; });
    document.querySelectorAll('[data-radar-sub2]').forEach((el) => { el.textContent = station.radarSub2 || ''; });

    // Update Today/Tonight row in 7-Day & 5-Day forecast cards so Golden/Night never show 70% rain Today
    document.querySelectorAll('[data-today-row]').forEach((row) => {
      const nameEl = row.querySelector('[data-today-name]');
      const iconEl = row.querySelector('[data-today-icon]');
      const popEl = row.querySelector('[data-today-pop]');
      const lowEl = row.querySelector('.day-low');
      const highEl = row.querySelector('.day-high');
      if (nameEl) nameEl.textContent = preset.todayRow.name;
      if (iconEl) iconEl.setAttribute('href', preset.todayRow.icon);
      if (popEl) popEl.textContent = preset.todayRow.pop || '\u00A0';
      if (lowEl) lowEl.dataset.f = String(preset.lowF);
      if (highEl) highEl.dataset.f = String(preset.highF);
    });

    // Update 7-day lows/highs & pinned station badges with [data-f]
    document.querySelectorAll('[data-f]').forEach((el) => {
      const baseF = Number(el.dataset.f);
      if (Number.isNaN(baseF)) return;
      const adjustedF = el.closest('[data-day-rows]') ? baseF + dF : baseF;
      el.textContent = fmtTemp(adjustedF);
    });

    // Refresh radar timeline timestamps for active weather regime
    applyRadarFrame(currentFrame);
  }

  function setWeather(presetKey) {
    if (!WEATHER_PRESETS[presetKey]) return;
    currentWeatherKey = presetKey;
    root.dataset.weather = presetKey;
    document.querySelectorAll('[data-set-weather]').forEach((btn) => {
      btn.setAttribute('aria-pressed', btn.dataset.setWeather === presetKey ? 'true' : 'false');
    });
    renderWeatherAndUnits();
  }

  function setUnits(units) {
    if (units !== 'imperial' && units !== 'metric') return;
    currentUnits = units;
    document.querySelectorAll('[data-unit-btn]').forEach((b) => {
      b.setAttribute('aria-pressed', b.dataset.unitBtn === units ? 'true' : 'false');
    });
    renderWeatherAndUnits();
  }

  function setStation(stationKey, customText) {
    if (customText) {
      const clean = customText.trim();
      if (!clean) return;
      const shortName = clean.split(',')[0].trim();
      customPlaceOverride = {
        place: clean,
        short: shortName,
        icao: 'KLOC',
        deltaF: 0,
        radarStation: `${shortName.toUpperCase()} ★`,
        radarNeighbor: 'REGIONAL',
        radarSub1: '',
        radarSub2: ''
      };
      document.querySelectorAll('[data-select-station]').forEach((b) => b.setAttribute('aria-pressed', 'false'));
    } else if (STATIONS[stationKey]) {
      currentStationKey = stationKey;
      customPlaceOverride = null;
      document.querySelectorAll('[data-select-station]').forEach((b) => {
        b.setAttribute('aria-pressed', b.dataset.selectStation === stationKey ? 'true' : 'false');
      });
      const input = document.getElementById('cfg-place-input');
      if (input) input.value = STATIONS[stationKey].place;
    }
    renderWeatherAndUnits();
  }

  // 4. Interactive iPhone Prototype Tab Dock (5-pill capsule + circular Config orb, roving tabindex + arrow keys)
  function setProtoTab(tabName, moveFocus = false) {
    const proto = document.getElementById('interactive-phone');
    if (!proto) return;
    const tabs = Array.from(proto.querySelectorAll('[data-proto-tab]'));
    let activeBtn = null;

    tabs.forEach((btn) => {
      const active = btn.dataset.protoTab === tabName;
      btn.classList.toggle('active', active);
      btn.setAttribute('aria-selected', active ? 'true' : 'false');
      btn.tabIndex = active ? 0 : -1;
      if (active) activeBtn = btn;
    });

    proto.querySelectorAll('[data-proto-screen]').forEach((scr) => {
      scr.hidden = scr.dataset.protoScreen !== tabName;
    });

    const phoneScreen = document.getElementById('interactive-phone-screen');
    const topBar = document.getElementById('proto-top-bar');
    if (phoneScreen) {
      phoneScreen.classList.toggle('radar-canvas', tabName === 'Radar');
      phoneScreen.classList.toggle('wx-atmosphere', tabName !== 'Radar');
    }
    if (topBar) {
      topBar.hidden = (tabName === 'Radar');
    }

    const readout = document.getElementById('proto-active-tab-label');
    if (readout) readout.textContent = tabName;

    if (moveFocus && activeBtn) {
      activeBtn.focus({ preventScroll: true });
    }
  }

  function initProtoDockKeyboard() {
    const proto = document.getElementById('interactive-phone');
    if (!proto) return;
    const tabs = Array.from(proto.querySelectorAll('[data-proto-tab]'));
    tabs.forEach((tab, idx) => {
      tab.addEventListener('click', () => setProtoTab(tab.dataset.protoTab, false));
      tab.addEventListener('keydown', (e) => {
        let nextIdx = null;
        if (e.key === 'ArrowRight') nextIdx = (idx + 1) % tabs.length;
        else if (e.key === 'ArrowLeft') nextIdx = (idx - 1 + tabs.length) % tabs.length;
        else if (e.key === 'Home') nextIdx = 0;
        else if (e.key === 'End') nextIdx = tabs.length - 1;
        if (nextIdx !== null) {
          e.preventDefault();
          setProtoTab(tabs[nextIdx].dataset.protoTab, true);
        }
      });
    });
  }

  // 5. Radar Scrubber & Play/Pause Control across Phone 2, Proto Phone, Mac Desk & iPad Split
  function syncSliderCSSVars(input) {
    input.style.setProperty('--attr-min', input.min || '0');
    input.style.setProperty('--attr-max', input.max || '5');
    input.style.setProperty('--control-value', input.value);
  }

  function applyRadarFrame(index) {
    currentFrame = ((index % RADAR_OFFSETS.length) + RADAR_OFFSETS.length) % RADAR_OFFSETS.length;
    const f = RADAR_OFFSETS[currentFrame];
    const preset = WEATHER_PRESETS[currentWeatherKey] || WEATHER_PRESETS.rain;
    const timeStr = preset.frameTimes[currentFrame] || preset.obsTime;

    document.querySelectorAll('[data-radar-echo-group]').forEach((g) => {
      g.setAttribute('transform', `translate(${f.shiftX}, ${f.shiftY}) scale(${f.scale})`);
    });
    document.querySelectorAll('[data-radar-time]').forEach((el) => {
      el.textContent = `${timeStr} · ${f.age}`;
    });
    const valueText = `Frame ${currentFrame + 1} of ${RADAR_OFFSETS.length}, ${timeStr} ${f.age}`;
    document.querySelectorAll('[data-radar-slider]').forEach((input) => {
      input.value = String(currentFrame);
      input.setAttribute('aria-valuetext', valueText);
      syncSliderCSSVars(input);
    });
  }

  function toggleRadarPlayback() {
    radarPlaying = !radarPlaying;
    document.querySelectorAll('[data-radar-play]').forEach((btn) => {
      btn.textContent = radarPlaying ? '❚❚ Pause' : '▶ Play';
      btn.setAttribute('aria-label', radarPlaying ? 'Pause radar loop' : 'Play radar loop');
    });
    if (radarPlaying) {
      radarTimer = window.setInterval(() => {
        applyRadarFrame(currentFrame + 1);
      }, 650);
    } else if (radarTimer) {
      window.clearInterval(radarTimer);
      radarTimer = null;
    }
  }

  // 6. Mac Menu Bar Popover Mode Switcher (Compact / Standard / Tactical)
  function setPopoverMode(mode) {
    document.querySelectorAll('[data-hud-mode]').forEach((btn) => {
      btn.setAttribute('aria-pressed', btn.dataset.hudMode === mode ? 'true' : 'false');
    });
    document.querySelectorAll('[data-hud-section]').forEach((sec) => {
      const allowed = (sec.dataset.hudSection || '').split(' ');
      sec.hidden = !allowed.includes(mode);
    });
  }

  // 7. macOS Desk Console 8-Mode Bridge Switcher (Tactical, Surface, Radar, Outlooks, Chase, Climate, Tropics, Grid)
  function setMacMode(mode) {
    if (!MAC_MODES.includes(mode)) return;
    currentMacMode = mode;
    document.querySelectorAll('[data-mac-mode]').forEach((b) => {
      const active = b.dataset.macMode === mode;
      b.setAttribute('aria-pressed', active ? 'true' : 'false');
    });
    document.querySelectorAll('[data-mac-pane]').forEach((pane) => {
      pane.hidden = pane.dataset.macPane !== mode;
    });
  }

  // Wire up controls + zero-flash URL query initialization
  document.addEventListener('DOMContentLoaded', () => {
    initStudioTabs();
    initProtoDockKeyboard();

    document.querySelectorAll('[data-set-direction]').forEach((btn) => {
      btn.addEventListener('click', () => withTransition(() => setDirection(btn.dataset.setDirection)));
    });

    document.querySelectorAll('[data-set-weather]').forEach((btn) => {
      btn.addEventListener('click', () => withTransition(() => setWeather(btn.dataset.setWeather)));
    });

    document.querySelectorAll('[data-open-config]').forEach((btn) => {
      btn.addEventListener('click', () => setProtoTab('Config'));
    });

    document.querySelectorAll('[data-jump-config]').forEach((btn) => {
      btn.addEventListener('click', () => {
        if (selectStudioTabById) selectStudioTabById('iphone-interactive');
        setProtoTab('Config');
      });
    });

    document.querySelectorAll('[data-jump-desk]').forEach((btn) => {
      btn.addEventListener('click', () => {
        if (selectStudioTabById) selectStudioTabById('mac-desk');
        setMacMode('Tactical');
      });
    });

    document.querySelectorAll('[data-jump-desk-radar]').forEach((btn) => {
      btn.addEventListener('click', () => {
        if (selectStudioTabById) selectStudioTabById('mac-desk');
        setMacMode('Radar');
      });
    });

    document.querySelectorAll('[data-open-severe]').forEach((btn) => {
      btn.addEventListener('click', () => setProtoTab('Severe'));
    });

    document.querySelectorAll('[data-radar-slider]').forEach((slider) => {
      syncSliderCSSVars(slider);
      slider.addEventListener('input', (e) => applyRadarFrame(Number(e.target.value)));
    });

    document.querySelectorAll('[data-radar-play]').forEach((btn) => {
      btn.addEventListener('click', toggleRadarPlayback);
    });

    document.querySelectorAll('[data-hud-mode]').forEach((btn) => {
      btn.addEventListener('click', () => setPopoverMode(btn.dataset.hudMode));
    });

    document.querySelectorAll('[data-mac-mode]').forEach((btn) => {
      btn.addEventListener('click', () => withTransition(() => setMacMode(btn.dataset.macMode)));
    });

    document.querySelectorAll('[data-chase-intercept]').forEach((btn) => {
      btn.addEventListener('click', () => withTransition(() => setMacMode('Radar')));
    });

    document.querySelectorAll('[data-grid-jump]').forEach((btn) => {
      btn.addEventListener('click', () => {
        setStation(btn.dataset.gridJump);
        withTransition(() => setMacMode('Tactical'));
      });
    });

    document.querySelectorAll('[data-unit-btn]').forEach((btn) => {
      btn.addEventListener('click', () => setUnits(btn.dataset.unitBtn));
    });

    document.querySelectorAll('[data-toggle-units]').forEach((btn) => {
      btn.addEventListener('click', () => setUnits(currentUnits === 'imperial' ? 'metric' : 'imperial'));
    });

    document.querySelectorAll('[data-select-station]').forEach((btn) => {
      btn.addEventListener('click', () => setStation(btn.dataset.selectStation));
    });

    const saveBtn = document.getElementById('cfg-save-btn');
    const gpsBtn = document.getElementById('cfg-gps-btn');
    const placeInput = document.getElementById('cfg-place-input');
    if (saveBtn && placeInput) {
      saveBtn.addEventListener('click', () => setStation(null, placeInput.value));
    }
    if (gpsBtn) {
      gpsBtn.addEventListener('click', () => setStation('newhaven'));
    }

    // Apply initial defaults + URL query parameters synchronously BEFORE enabling transitions
    renderWeatherAndUnits();
    applyRadarFrame(5);

    const params = new URLSearchParams(window.location.search);
    if (params.get('station')) setStation(params.get('station'));
    if (params.get('units')) setUnits(params.get('units'));
    if (params.get('dir')) setDirection(params.get('dir'));
    if (params.get('wx')) setWeather(params.get('wx'));
    if (params.get('proto')) setProtoTab(params.get('proto'));
    if (params.get('hud')) setPopoverMode(params.get('hud'));
    if (params.get('mac')) setMacMode(params.get('mac'));
    if (params.get('tab') && selectStudioTabById) {
      selectStudioTabById(params.get('tab'), true);
    }

    // Force style flush on frame 0 before arming interactive transitions
    void document.body.offsetHeight;
    window.requestAnimationFrame(() => {
      window.requestAnimationFrame(() => {
        root.dataset.ready = 'true';
      });
    });

    // Expose clean API for programmatic verification
    window.wxMocks = {
      setDirection,
      setWeather,
      setUnits,
      setStation,
      setProtoTab,
      setPopoverMode,
      setMacMode,
      applyRadarFrame,
      selectStage(id) {
        if (selectStudioTabById) selectStudioTabById(id, true);
      }
    };
  });
})();
