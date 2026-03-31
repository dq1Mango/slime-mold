const PARTICLE_COUNT = 1;
const FLOATS_PER_PARTICLE = 2;

const canvas = document.getElementById("canvas");
const gl = canvas.getContext("webgl2");
const tf = gl.createTransformFeedback();

function compileShader(gl, type, source) {
  const shader = gl.createShader(type);
  gl.shaderSource(shader, source);
  gl.compileShader(shader);
  if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
    console.error(gl.getShaderInfoLog(shader));
    gl.deleteShader(shader);
    return null;
  }
  return shader;
}

function createProgram(gl, vertSrc, fragSrc) {
  const vert = compileShader(gl, gl.VERTEX_SHADER, vertSrc);
  const frag = compileShader(gl, gl.FRAGMENT_SHADER, fragSrc);
  const program = gl.createProgram();
  gl.attachShader(program, vert);
  gl.attachShader(program, frag);
  gl.linkProgram(program);
  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    console.error(gl.getProgramInfoLog(program));
    return null;
  }
  return program;
}

async function loadShader(path) {
  const res = await fetch(path);
  return res.text();
}

// some secret sauce claude cooked up that i cant be bothered to care abt
const observer = new ResizeObserver((entries) => {
  for (const entry of entries) {
    const width =
      entry.devicePixelContentBoxSize?.[0].inlineSize ??
      entry.contentBoxSize[0].inlineSize * devicePixelRatio;
    const height =
      entry.devicePixelContentBoxSize?.[0].blockSize ??
      entry.contentBoxSize[0].blockSize * devicePixelRatio;

    canvas.width = Math.round(width);
    canvas.height = Math.round(height);

    // update the WebGL viewport to match
    gl.viewport(0, 0, canvas.width, canvas.height);

    // draw(3);
  }
});

observer.observe(canvas, { box: "device-pixel-content-box" });

let initialized = false;
function draw(length) {
  gl.drawArrays(gl.POINTS, 0, length);
}

async function test() {
  // const [simVertSrc, renderVertSrc, renderFragSrc] = await Promise.all([
  //   loadShader("shaders/sim.vert"),
  //   loadShader("shaders/render.vert"),
  //   loadShader("shaders/render.frag"),
  // ]);
  //
  const [testVertSrc, testFragSrc] = await Promise.all([
    loadShader("shaders/test.vert"),
    loadShader("shaders/test.frag"),
  ]);
  //
  // const simulationProgram = gl.createProgram();
  //
  // const simVert = compileShader(gl, gl.VERTEX_SHADER, simVertSrc);
  // gl.attachShader(simulationProgram, simVert);

  //const renderProgram = createProgram(gl, renderVertSrc, renderFragSrc);
  //

  // testing

  const testProgram = createProgram(gl, testVertSrc, testFragSrc);
  gl.useProgram(testProgram);

  const aPosition = 0;
  const aSize = 1;
  const aColor = 2;

  let dataData = [
    [0, 0, 50, 1, 0, 0],
    [0.5, 0, 50, 0, 1, 0],
    [0, 0, 100, 0, 0, 1],
  ];

  let iSuckAtJS = [];

  for (data of dataData) {
    // console.log(data);
    iSuckAtJS = iSuckAtJS.concat(data);
  }
  // console.log(iSuckAtJS);
  let vertData = new Float32Array(iSuckAtJS);

  // console.log(vertData);

  const vertBuffer = gl.createBuffer();

  gl.bindBuffer(gl.ARRAY_BUFFER, vertBuffer);
  gl.bufferData(gl.ARRAY_BUFFER, vertData, gl.STATIC_DRAW);

  const STRIDE = 2 * 4 + 1 * 4 + 3 * 4;

  gl.vertexAttribPointer(aPosition, 2, gl.FLOAT, false, STRIDE, 0);
  gl.vertexAttribPointer(aSize, 1, gl.FLOAT, false, STRIDE, 2 * 4);
  gl.vertexAttribPointer(aColor, 3, gl.FLOAT, false, STRIDE, 2 * 4 + 4);

  gl.enableVertexAttribArray(aPosition);
  gl.enableVertexAttribArray(aColor);
  gl.enableVertexAttribArray(aSize);

  gl.enable(gl.BLEND);
  gl.blendFunc(gl.SRC_ALPHA, gl.ONE);
  // gl.blendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA);

  initialized = true;
  draw(dataData.length);
  // gl.drawArrays(gl.POINTS, 0, dataData.length);
}

const vertData = new Float32Array([0, 0]);

const bufA = gl.createBuffer();

gl.bindBuffer(gl.ARRAY_BUFFER, bufA);
gl.bufferData(gl.ARRAY_BUFFER, vertData, gl.DYNAMIC_COPY);

const bufB = gl.createBuffer();

gl.bindBuffer(gl.ARRAY_BUFFER, bufB);
gl.bufferData(gl.ARRAY_BUFFER, vertData.length * 4, gl.DYNAMIC_COPY);

let read = bufA,
  write = bufB;

function makeVAO(program, buffer) {
  const vao = gl.createVertexArray();
  gl.bindVertexArray(vao);
  gl.bindBuffer(gl.ARRAY_BUFFER, buffer);

  // const stride = FLOATS_PER_PARTICLE * 4;
  const stride = 0;
  // const posLoc = gl.getAttribLocation(program, "inPosition");
  const posLoc = 0;
  // const velLoc = gl.getAttribLocation(program, 'inVelocity');
  // const rngLoc = gl.getAttribLocation(program, 'inRngState');

  gl.vertexAttribPointer(posLoc, 2, gl.FLOAT, false, stride, 0);
  gl.enableVertexAttribArray(posLoc);
  // gl.enableVertexAttribArray(velLoc);
  // gl.vertexAttribPointer(velLoc, 2, gl.FLOAT, false, stride, 8);
  // gl.enableVertexAttribArray(rngLoc);
  // gl.vertexAttribPointer(rngLoc, 1, gl.FLOAT, false, stride, 16);

  gl.bindVertexArray(null);
  return vao;
}

let vaoA;
let vaoB;

let count = 0;

function frame() {
  if (!initialized) {
    return;
  }

  if (count > 32) {
    return;
  }

  gl.bindVertexArray(read === bufA ? vaoA : vaoB);
  gl.bindTransformFeedback(gl.TRANSFORM_FEEDBACK, tf);

  console.log(bufA, bufB); // should be different numbers e.g. 1 and 2
  gl.bindBufferBase(gl.TRANSFORM_FEEDBACK_BUFFER, 0, write);
  gl.enable(gl.RASTERIZER_DISCARD); // no fragment shader needed
  gl.beginTransformFeedback(gl.POINTS);

  gl.drawArrays(gl.POINTS, 0, PARTICLE_COUNT);
  gl.endTransformFeedback();
  gl.disable(gl.RASTERIZER_DISCARD);
  gl.bindTransformFeedback(gl.TRANSFORM_FEEDBACK, null);

  // render pass
  // gl.useProgram(renderProgram);
  // gl.bindVertexArray(read === bufA ? vaoA : vaoB); // render from write buffer
  // gl.drawArrays(gl.POINTS, 0, PARTICLE_COUNT);
  //
  // swap
  [read, write] = [write, read];

  count++;

  requestAnimationFrame(frame);
}

async function main() {
  const [simVertSrc, renderVertSrc, renderFragSrc] = await Promise.all([
    loadShader("shaders/sim.vert"),
    loadShader("shaders/render.vert"),
    loadShader("shaders/render.frag"),
  ]);

  const nullFrag = compileShader(
    gl,
    gl.FRAGMENT_SHADER,
    `#version 300 es
precision highp float;
void main() {}`,
  );

  //
  //

  const simulationProgram = gl.createProgram();
  // gl.useProgram(simulationProgram);
  const simVert = compileShader(gl, gl.VERTEX_SHADER, simVertSrc);
  gl.attachShader(simulationProgram, simVert);
  gl.attachShader(simulationProgram, nullFrag);

  gl.transformFeedbackVaryings(simulationProgram, ["outPosition"], gl.INTERLEAVED_ATTRIBS);

  gl.linkProgram(simulationProgram);

  vaoA = makeVAO(simulationProgram, bufA);
  vaoB = makeVAO(simulationProgram, bufB);

  // let vao = makeVAO(simulationProgram, write);

  gl.useProgram(simulationProgram);

  initialized = true;

  requestAnimationFrame(frame);

  // const renderProgram = createProgram(gl, renderVertSrc, renderFragSrc);
}

main();
