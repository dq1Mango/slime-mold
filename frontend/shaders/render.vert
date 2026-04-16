#version 300 es

in vec4 inPosition;

void main() {
    gl_Position = inPosition;
    gl_PointSize = 100.0 * inPosition.x;

    if (gl_PointSize < 0.0) {
        gl_PointSize *= -1.0;
    }
}
