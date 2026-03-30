#version 300 es

layout(location = 0) in vec4 aPosition;
layout(location = 1) in float aSize;
layout(location = 2) in vec4 aColor;

out vec4 vColor;

void main() {
    gl_Position = aPosition;
    gl_PointSize = aSize;

    vColor = aColor;
}
