#version 300 es

layout(location = 0) in vec4 inPosition;

out vec4 outPosition;

void main() {
    outPosition = inPosition;
    outPosition.x += 0.01;

    if (outPosition.x > 1.0) {
        outPosition.x = -1.0;
    }
}
