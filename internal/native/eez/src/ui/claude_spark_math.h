// Pure integer maths of the breathing Claude spark (no LVGL, host-testable, see eez/test).
//
// Wave: ray i has the phase (t * 360 / PERIOD_MS) - i * 30 degrees, so the phase of neighbouring
// rays differs by 1/12 of a round. Length in tenths of a pixel: 135 + 25 * sin(phase), i.e. 110..160
// (11 to 16 px, middle 13.5, +-2.5 px).
// Worked examples (sin as lv_trigo_sin, -32767..32767): phase 0 -> sin 0 -> 135 (13.5 px);
// phase 90 -> 32767 -> 160 (16 px); phase 270 -> -32767 -> 110 (11 px); phase 30 -> 16383 -> 147.
#ifndef CLAUDE_SPARK_MATH_H
#define CLAUDE_SPARK_MATH_H

#define CLAUDE_SPARK_PERIOD_MS 4000
#define CLAUDE_SPARK_LEN_MID10 135
#define CLAUDE_SPARK_LEN_AMP10 25

// Ray length in 1/10 px from the sine value of its phase (-32768..32767).
static inline int claude_spark_len10(int sin_val) {
    if (sin_val > 32767) sin_val = 32767;
    if (sin_val < -32767) sin_val = -32767;
    return CLAUDE_SPARK_LEN_MID10 + CLAUDE_SPARK_LEN_AMP10 * sin_val / 32767;
}

// Phase in degrees (0..359) of ray i at time t_ms.
static inline int claude_spark_phase(unsigned t_ms, int i) {
    int base = (int)((t_ms % CLAUDE_SPARK_PERIOD_MS) * 360u / CLAUDE_SPARK_PERIOD_MS);
    int p = (base - i * 30) % 360;
    return p < 0 ? p + 360 : p;
}

// Offset of a ray tip: dir1000 (direction * 1000) times len10 (1/10 px), rounded to whole pixels.
static inline int claude_spark_offset(int dir1000, int len10) {
    int v = dir1000 * len10;
    return (v + (v >= 0 ? 5000 : -5000)) / 10000;
}

#endif
