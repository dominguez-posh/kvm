// Host test: gcc -I../src/ui claude_spark_test.c -lm -o /tmp/spark_test && /tmp/spark_test
#include <math.h>
#include <stdio.h>
#include <stdlib.h>
#include "claude_spark_math.h"

static int fails;
#define CHECK(c) do { if (!(c)) { printf("FAIL line %d: %s\n", __LINE__, #c); fails++; } } while (0)

static int sin15(int deg) { return (int)lround(sin(deg * M_PI / 180.0) * 32767); }

int main(void) {
    CHECK(claude_spark_len10(0) == 135);
    CHECK(claude_spark_len10(32767) == 160);
    CHECK(claude_spark_len10(-32767) == 110);
    CHECK(claude_spark_len10(-32768) == 110);
    CHECK(claude_spark_len10(16383) == 147);
    CHECK(claude_spark_phase(0, 0) == 0);
    CHECK(claude_spark_phase(0, 1) == 330);
    CHECK(claude_spark_phase(1000, 0) == 90);
    CHECK(claude_spark_phase(4000, 0) == 0);
    CHECK(claude_spark_phase(1u << 31, 11) < 360);
    CHECK(claude_spark_offset(1000, 160) == 16);
    CHECK(claude_spark_offset(-1000, 160) == -16);
    CHECK(claude_spark_offset(866, 135) == 12);
    CHECK(claude_spark_offset(-866, 135) == -12);
    // all rays, whole round: length 11..16 px, tip + half line width (2) inside the 38 px box (centre 19)
    for (unsigned t = 0; t < 8000; t += 33) {
        for (int i = 0; i < 12; i++) {
            int l = claude_spark_len10(sin15(claude_spark_phase(t, i)));
            CHECK(l >= 110 && l <= 160);
            int dx = (int)lround(cos(i * 30 * M_PI / 180) * 1000), dy = (int)lround(sin(i * 30 * M_PI / 180) * 1000);
            int x = 19 + claude_spark_offset(dx, l), y = 19 + claude_spark_offset(dy, l);
            CHECK(x - 2 >= 0 && x + 2 <= 38 && y - 2 >= 0 && y + 2 <= 38);
        }
    }
    // neighbouring rays are 1/12 round apart
    CHECK(claude_spark_phase(500, 0) - claude_spark_phase(500, 1) == 30);
    printf(fails ? "FAILED\n" : "OK\n");
    return fails != 0;
}
