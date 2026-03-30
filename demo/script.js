const canvas = document.getElementById("canvas");
const gl = canvas.getContext("webgl2");

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
  }
});

observer.observe(canvas, { box: "device-pixel-content-box" });

async function main() {
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

  gl.vertexAttrib4f(aPosition, 0, 0, 0, 1);
  gl.vertexAttrib1f(aSize, 50);
  gl.vertexAttrib4f(aColor, 1, 0, 0, 1);

  gl.drawArrays(gl.POINTS, 0, 1);
}

main();
