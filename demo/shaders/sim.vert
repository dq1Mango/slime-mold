#version 300 es

layout(location = 0) in vec4 inPosition;

out vec4 outPosition;

void main() {
    outPosition = inPosition;
    outPosition[0] += 0.01;
}
