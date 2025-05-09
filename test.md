Creating a Base64 encoder using intrinsics for optimal performance can significantly enhance processing speed, especially when dealing with large volumes of data. Below is an implementation that leverages SIMD (Single Instruction, Multiple Data) operations using Intel's AVX2 intrinsics. This implementation will work efficiently to encode data into Base64 format.

Ensure your compiler supports the necessary intrinsics by including the appropriate headers and enabling the required flags (e.g., `-mavx2` for GCC or Clang).

### Base64 Encoding in C Using AVX2 Intrinsics

```c
#include <stdio.h>
#include <stdint.h>
#include <immintrin.h> // Header for AVX and other SIMD intrinsics

static const char base64_table[65] =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    "abcdefghijklmnopqrstuvwxyz"
    "0123456789+/";

void base64_encode(const uint8_t *input, size_t length, char *output) {
    size_t i, j;
    size_t output_length = 4 * ((length + 2) / 3); // Calculate output size
    if (output_length % 4 != 0) {
        output_length += 4 - (output_length % 4);
    }
    
    for (i = 0, j = 0; i < length;) {
        // Read 3 bytes
        __m128i in = _mm_setzero_si128();
        if (i < length) in = _mm_set_epi8(i + 2 < length ? input[i + 2] : 0, i + 1 < length ? input[i + 1] : 0, input[i]);
        i += 3;

        // Create 4 bytes out of 3 bytes (24 bits)
        __m128i temp = _mm_packus_epi16(_mm_set_epi16(0, 0, 0, 0, _mm_extract_epi8(in, 0), _mm_extract_epi8(in, 1), _mm_extract_epi8(in, 2), 0), _mm_set_epi16(0, 0, 0, 0, _mm_extract_epi8(in, 3), _mm_extract_epi8(in, 4), _mm_extract_epi8(in, 5), 0));

        // Map 24 bits of input to 4 bytes of output
        __m128i encoded = _mm_set_epi8(
            base64_table[_mm_extract_epi8(temp, 0) & 0x3F],
            base64_table[(_mm_extract_epi8(temp, 0) >> 6) | ((_mm_extract_epi8(temp, 1) & 0x03) << 2)],
            base64_table[(_mm_extract_epi8(temp, 1) >> 2) & 0x3F],
            base64_table[(_mm_extract_epi8(temp, 1) >> 4) | ((_mm_extract_epi8(temp, 2) & 0x0F) << 4)]
        );

        // Store the Base64 encoded output
        for (int k = 0; k < 4; k++) {
            output[j++] = _mm_extract_epi8(encoded, k);
        }
    }

    // Handle padding
    for (size_t k = output_length; k > j; k--) {
        output[k - 1] = '=';
    }

    output[output_length] = '\0'; // Null-terminate the output string
}

int main() {
    const char *input = "Hello, World!";  // Example input
    size_t input_length = strlen(input);
    size_t output_length = 4 * ((input_length + 2) / 3);

    char *output = (char *)malloc(output_length + 1); // Extra space for null terminator

    base64_encode((uint8_t *)input, input_length, output);

    printf("Original: %s\n", input);
    printf("Encoded: %s\n", output);

    free(output);
    return 0;
}
```

### Explanation:
1. **Constants**: A lookup table for the Base64 encoding characters is created.
2. **Function `base64_encode`**:
   - Reads input in chunks of 3 bytes.
   - Utilizes intrinsics to efficiently process 24-bit segments and creates appropriately mapped Base64 transform characters.
   - Pads the output with `=` characters when the input length is not a multiple of 3.
3. **Main Function**: Tests the encoder with a simple string.

### Notes:
- The above implementation is simplified for illustrative purposes and may not handle all corner cases robustly.
- Make sure to account for padding correctly and verify output with various input sizes.
- Optimization may vary depending on hardware architecture; consider benchmarking on the target CPU.
- You can further refine the use of intrinsic functions based on the specific use case and performance requirements.

