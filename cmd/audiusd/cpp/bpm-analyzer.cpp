// bpm_analyzer.cpp
#include <aubio/aubio.h>
#include <iostream>
#include <string>
#include <vector>
#include <map>
#include <iomanip>
#include <algorithm>

float compute_median(const std::vector<float>& bpms) {
    if (bpms.empty()) {
        return 0.0f;
    }

    std::vector<float> sorted_bpms = bpms;
    std::sort(sorted_bpms.begin(), sorted_bpms.end());

    size_t size = sorted_bpms.size();
    if (size % 2 == 0) {
        return (sorted_bpms[size/2 - 1] + sorted_bpms[size/2]) / 2.0f;
    } else {
        return sorted_bpms[size/2];
    }
}

float analyze_bpm(const char* filename) {
    uint_t win_s = 2048;           // window size
    uint_t hop_s = 512;            // hop size
    unsigned int samplerate = 0;   // samplerate will be set by source

    aubio_source_t* source = new_aubio_source(filename, samplerate, hop_s);
    if (!source) {
        std::cerr << "Error: could not open " << filename << std::endl;
        return 0;
    }

    samplerate = aubio_source_get_samplerate(source);

    // aubio's energy-based modeling is most similar to essentia's `RhythmExtractor2013` 
    aubio_tempo_t* tempo = new_aubio_tempo("energy", win_s, hop_s, samplerate);
    if (!tempo) {
        std::cerr << "Error: could not create tempo object" << std::endl;
        del_aubio_source(source);
        return 0;
    }

    fvec_t* in = new_fvec(hop_s);
    fvec_t* out = new_fvec(2);

    uint_t read = 0;
    float current_time = 0.0f;
    std::vector<float> detected_bpms;

    do {
        aubio_source_do(source, in, &read);
        current_time += static_cast<float>(read) / samplerate;

        aubio_tempo_do(tempo, in, out);

        // Skip detections in first 5 seconds (fade-ins, silence)
        if (out->data[0] != 0 && current_time > 5.0f) {
            float current_bpm = aubio_tempo_get_bpm(tempo);
            // Skip detections outside of 40-200 BPM range
            if (current_bpm >= 40 && current_bpm <= 200) {
                detected_bpms.push_back(current_bpm);
            }
        }
    } while (read == hop_s);

    float final_bpm = 0.0f;

    if (!detected_bpms.empty()) {
        final_bpm = compute_median(detected_bpms);
    }

    // Cleanup
    del_fvec(in);
    del_fvec(out);
    del_aubio_tempo(tempo);
    del_aubio_source(source);

    return final_bpm;
}

int main(int argc, char** argv) {
    if (argc != 2) {
        std::cerr << "Usage: " << argv[0] << " <filename>" << std::endl;
        return 1;
    }

    float bpm = analyze_bpm(argv[1]);

    if (bpm > 0) {
        std::cout << std::fixed << std::setprecision(2);
        std::cout << "Estimated BPM: " << bpm << std::endl;
        return 0;
    }

    std::cerr << "Could not estimate BPM." << std::endl;
    return 1;
}
