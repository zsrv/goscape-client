// IconDump.java — Java reference harness for the goscape-client item-icon
// rasterizer, rev-254 branch.
//
// Renders rev-254 golden pixels straight from the PINNED, byte-for-byte
// LostCityRS/Client-Java render code (worktree checked out at commit
// 2e629784, branch lineage of Server254-ref/javaclient). Every golden this
// file writes is the single source of truth the Go item-icon port diffs
// against, so this harness calls the pinned client routines EXACTLY as the real
// client does and never re-implements any render math.
//
// It is the Java-254 analogue of goscape-client rev-274's TypeScript harness
// tools/iconref/dump.ts (which drove the vendored Client-TS). The pipeline,
// init order, category buckets and sample-selection algorithm are ported from
// that file so the two branches' goldens are produced the same way.
//
// Outputs (all under --out, default ../../cmd/icondump/testdata):
//   palette.bin              65536 x int32 LE  (Pix3D.colourTable @ 0.8)
//   tri/*.rgba + manifest    synthetic single-triangle goldens (8 cases)
//   icons254/*.rgba + *.png  the curated item icons via ObjType.getSprite
//   icons254/sample.json     the curated sample (id, debugname, categories)
//
// Build/run: see build.sh (compiles the pinned client + this file, applying the
// documented Math.random neutralisation patch, then runs against the ref cache).
//
// DETERMINISM NOTE (critical): Pix3D.initColourTable() jitters brightness by
// `Math.random() * 0.03 - 0.015`. A golden must be reproducible, so build.sh
// patches the $TMPDIR copy of Pix3D.java to drop the jitter term (var3 = arg1),
// making initColourTable(0.8) build the palette at brightness 0.8 to the bit.
// This mirrors dump.ts pinning Math.random => 0.5 (0.5*0.03-0.015 == 0). It is
// the ONE intentional deviation from a literal client run and removes
// randomness rather than adding behaviour. The Go port builds its palette with
// the un-jittered 0.8 brightness (InitColourTableDeterministic(0.8)) to match.

import java.io.ByteArrayInputStream;
import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.TreeSet;
import java.util.zip.GZIPInputStream;

import javax.imageio.ImageIO;
import java.awt.image.BufferedImage;

import jagex2.config.ObjType;
import jagex2.dash3d.Metadata;
import jagex2.dash3d.Model;
import jagex2.graphics.Pix2D;
import jagex2.graphics.Pix3D;
import jagex2.graphics.Pix32;
import jagex2.io.JagFile;
import jagex2.io.OnDemandProvider;

public final class IconDump {
    // 254 obj ids of the two mandatory sample items (Server254-ref/content/pack/obj.pack).
    static final int BRONZE_DAGGER = 1205;
    static final int KNIFE = 946;

    // ---- args -----------------------------------------------------------
    static String argVal(String[] a, String name, String def) {
        for (int i = 0; i + 1 < a.length; i++) {
            if (a[i].equals(name)) return a[i + 1];
        }
        return def;
    }

    // ---- classic .dat/.idx cache reader (ported from dump.ts) -----------
    // idx entry = size(3 BE) + firstSector(3 BE); each 520-byte dat sector =
    // 8-byte header [file(2), part(2), nextSector(3), archive+1(1)] + up to 512
    // payload bytes. Archive 0 (config jags) is raw; archives 1-4 are gzip/file.
    static final int SECTOR = 520;
    static final int SECTOR_DATA = 512;

    static final class Cache {
        final byte[] dat;
        final byte[][] idx = new byte[5][];

        Cache(String dir) throws IOException {
            dat = Files.readAllBytes(Path.of(dir, "main_file_cache.dat"));
            for (int i = 0; i <= 4; i++) {
                idx[i] = Files.readAllBytes(Path.of(dir, "main_file_cache.idx" + i));
            }
        }

        int count(int archive) {
            return archive < 0 || archive > 4 ? 0 : idx[archive].length / 6;
        }

        byte[] readRaw(int archive, int file) {
            if (archive < 0 || archive > 4) return null;
            byte[] ix = idx[archive];
            if (file < 0 || file * 6 + 6 > ix.length) return null;
            int p = file * 6;
            int size = ((ix[p] & 0xff) << 16) | ((ix[p + 1] & 0xff) << 8) | (ix[p + 2] & 0xff);
            int sector = ((ix[p + 3] & 0xff) << 16) | ((ix[p + 4] & 0xff) << 8) | (ix[p + 5] & 0xff);
            if (size <= 0 || size > 2_000_000) return null;
            int maxSector = dat.length / SECTOR;
            if (sector <= 0 || sector > maxSector) return null;

            byte[] out = new byte[size];
            int pos = 0;
            for (int part = 0; pos < size; part++) {
                if (sector == 0) break;
                int base = sector * SECTOR;
                int sFile = ((dat[base] & 0xff) << 8) | (dat[base + 1] & 0xff);
                int sPart = ((dat[base + 2] & 0xff) << 8) | (dat[base + 3] & 0xff);
                int next = ((dat[base + 4] & 0xff) << 16) | ((dat[base + 5] & 0xff) << 8) | (dat[base + 6] & 0xff);
                int sIdx = dat[base + 7] & 0xff;
                if (file != sFile || part != sPart || archive != sIdx - 1) {
                    throw new RuntimeException("cache: sector header mismatch archive=" + archive + " file=" + file + " part=" + part);
                }
                if (next < 0 || next > maxSector) return null;
                int avail = Math.min(SECTOR_DATA, size - pos);
                System.arraycopy(dat, base + 8, out, pos, avail);
                pos += avail;
                sector = next;
            }
            return out;
        }

        // Full read: gunzip archives 1-4 (GZIPInputStream stops at the gzip
        // member end, ignoring the 2-byte version trailer), return archive 0 raw.
        byte[] read(int archive, int file) throws IOException {
            byte[] raw = readRaw(archive, file);
            if (raw == null) return null;
            if (archive == 0) return raw;
            try (GZIPInputStream gz = new GZIPInputStream(new ByteArrayInputStream(raw))) {
                return gz.readAllBytes();
            }
        }
    }

    // ---- RGBA conversion (mirrors dump.ts / the Go port) ----------------
    // A Pix32/Pix2D buffer holds 0x00RRGGBB ints; 0 is the client's "not drawn"
    // sentinel. p==0 -> (0,0,0,0); p!=0 -> (R,G,B,255) from the low 24 bits.
    static byte[] toRGBA(int[] pix, int w, int h) {
        byte[] out = new byte[w * h * 4];
        for (int i = 0; i < w * h; i++) {
            int q = pix[i];
            if (q == 0) continue;
            out[i * 4] = (byte) ((q >>> 16) & 0xff);
            out[i * 4 + 1] = (byte) ((q >>> 8) & 0xff);
            out[i * 4 + 2] = (byte) (q & 0xff);
            out[i * 4 + 3] = (byte) 0xff;
        }
        return out;
    }

    // Eyeball-only PNG (the .rgba is the golden). ARGB with the same transparency rule.
    static void writePNG(File file, int[] pix, int w, int h) throws IOException {
        BufferedImage img = new BufferedImage(w, h, BufferedImage.TYPE_INT_ARGB);
        for (int y = 0; y < h; y++) {
            for (int x = 0; x < w; x++) {
                int q = pix[y * w + x];
                int argb = q == 0 ? 0 : (0xff000000 | (q & 0xffffff));
                img.setRGB(x, y, argb);
            }
        }
        ImageIO.write(img, "png", file);
    }

    static void writeGolden(Path dir, String name, byte[] rgba) throws IOException {
        Files.write(dir.resolve(name), rgba);
    }

    // ---- main -----------------------------------------------------------
    public static void main(String[] args) throws Exception {
        String cacheDir = argVal(args, "--cache", "/home/owner/Code/github.com/LostCityRS/Server254-ref/unpack-ref/cache");
        String outDir = argVal(args, "--out", "../../cmd/icondump/testdata");
        String objPack = argVal(args, "--objpack", "/home/owner/Code/github.com/LostCityRS/Server254-ref/content/pack/obj.pack");

        Path out = Path.of(outDir);
        Files.createDirectories(out);
        Cache cache = new Cache(cacheDir);
        System.out.println("cache: " + cacheDir);
        System.out.printf("idx counts: idx0=%d idx1=%d idx2=%d idx3=%d idx4=%d%n",
                cache.count(0), cache.count(1), cache.count(2), cache.count(3), cache.count(4));

        // config jag (archive 0, file 2) -> ObjType.unpack
        byte[] configBytes = cache.read(0, 2);
        if (configBytes == null) throw new RuntimeException("cannot read config jag (0,2)");
        ObjType.unpack(new JagFile(configBytes));
        System.out.println("ObjType.unpack OK — count=" + ObjType.count);

        // textures (archive 0, file 6) then colour table (Client init order).
        // lowMem = false (the Go port forces LowMem = false) selects getTexels'
        // 128x128 / 64->128-upsampling path + 65536-entry texel pool. Java 254
        // defaults lowMem = true, so set it explicitly BEFORE unpackTextures /
        // initColourTable / initPool, all of which branch on it.
        Pix3D.lowMem = false;
        byte[] texBytes = cache.read(0, 6);
        if (texBytes == null) throw new RuntimeException("cannot read textures jag (0,6)");
        Pix3D.unpackTextures(new JagFile(texBytes));
        // Pix3D.initColourTable's Math.random jitter is patched out in the
        // $TMPDIR client copy (see header / build.sh) so 0.8 is used verbatim.
        Pix3D.initColourTable(0.8);
        Pix3D.initPool(20);
        System.out.println("Pix3D.unpackTextures + initColourTable(0.8) [jitter patched out] + initPool(20) OK");

        // pre-unpack every archive-1 model so Model.load never hits the provider.
        int numModels = cache.count(1);
        Model.init(numModels, new OnDemandProvider() {
            @Override public void requestModel(int id) { }
        });
        int unpacked = 0;
        for (int id = 0; id < numModels; id++) {
            byte[] raw;
            try {
                raw = cache.read(1, id);
            } catch (Exception e) {
                raw = null;
            }
            if (raw == null) continue;
            try {
                Model.unpack(raw, id); // Java 254: unpack(byte[] data, int id)
                unpacked++;
            } catch (Exception e) {
                // leave meta unset; Model.load(id) returns null for this id
            }
        }
        System.out.println("models: unpacked " + unpacked + "/" + numModels + " archive-1 files");

        // debugname map from the RuneScript obj.pack symbol table.
        Map<Integer, String> nameById = new HashMap<>();
        for (String line : Files.readString(Path.of(objPack)).split("\n")) {
            String t = line.trim();
            if (t.isEmpty()) continue;
            int eq = t.indexOf('=');
            if (eq < 0) continue;
            nameById.put(Integer.parseInt(t.substring(0, eq)), t.substring(eq + 1));
        }

        dumpPalette(out);
        dumpTriangles(out);
        dumpIcons(out, nameById);
        System.out.println("DONE");
    }

    // (a) palette.bin — 65536 int32 little-endian
    static void dumpPalette(Path out) throws IOException {
        byte[] buf = new byte[65536 * 4];
        for (int i = 0; i < 65536; i++) {
            int v = Pix3D.colourTable[i];
            buf[i * 4] = (byte) (v & 0xff);
            buf[i * 4 + 1] = (byte) ((v >>> 8) & 0xff);
            buf[i * 4 + 2] = (byte) ((v >>> 16) & 0xff);
            buf[i * 4 + 3] = (byte) ((v >>> 24) & 0xff);
        }
        Files.write(out.resolve("palette.bin"), buf);
        System.out.println("palette.bin: " + buf.length + " bytes (expect 262144)");
    }

    // (b) synthetic single-triangle goldens. Inputs mirror dump.ts exactly; the
    // Java argument order is y-major-first (the Go port mirrors it), so the same
    // named args map into Java positions as the Go golden test replays them.
    static void dumpTriangles(Path out) throws IOException {
        Path dir = out.resolve("tri");
        Files.createDirectories(dir);
        final int W = 32, H = 32;
        StringBuilder manifest = new StringBuilder("[\n");
        boolean first = true;

        // helper to set the render state dump.ts sets before each draw.
        // (declared inline per-case below)

        String[] names = {
                "gouraud_small", "gouraud_large", "gouraud_degenerate", "flat",
                "gouraud_alpha128", "textured_tex1", "textured_shaded", "textured_trans7"
        };
        for (String name : names) {
            int prefill = name.equals("gouraud_alpha128") ? 0x303030 : 0;
            boolean lowDetail = false, hclip = false;
            int trans = name.equals("gouraud_alpha128") ? 128 : 0;
            String routine;
            String argsJson;

            int[] px = new int[W * H];
            for (int i = 0; i < px.length; i++) px[i] = prefill;
            Pix2D.setPixels(W, px, H); // (width, pixels, height)
            Pix3D.init();              // == TS setRenderClipping (originX/Y = 16)
            Pix3D.lowDetail = lowDetail;
            Pix3D.hclip = hclip;
            Pix3D.trans = trans;

            switch (name) {
                case "gouraud_small" -> {
                    routine = "gouraud";
                    // xA12 xB22 xC15 yA10 yB12 yC24
                    Pix3D.gouraudTriangle(10, 12, 24, 12, 22, 15, 0x107f, 0x287f, 0x407f);
                    argsJson = gouraudArgs(12, 22, 15, 10, 12, 24, 0x107f, 0x287f, 0x407f);
                }
                case "gouraud_large" -> {
                    routine = "gouraud";
                    // xA1 xB30 xC4 yA1 yB6 yC30
                    Pix3D.gouraudTriangle(1, 6, 30, 1, 30, 4, 0x087f, 0x307f, 0x607f);
                    argsJson = gouraudArgs(1, 30, 4, 1, 6, 30, 0x087f, 0x307f, 0x607f);
                }
                case "gouraud_degenerate" -> {
                    routine = "gouraud";
                    // collinear: xA4 xB16 xC28 yA4 yB16 yC28
                    Pix3D.gouraudTriangle(4, 16, 28, 4, 16, 28, 0x107f, 0x307f, 0x507f);
                    argsJson = gouraudArgs(4, 16, 28, 4, 16, 28, 0x107f, 0x307f, 0x507f);
                }
                case "flat" -> {
                    routine = "flat";
                    // xA4 xB28 xC12 yA6 yB4 yC30 colour ff8040
                    Pix3D.flatTriangle(6, 4, 30, 4, 28, 12, 0xff8040);
                    argsJson = flatArgs(4, 28, 12, 6, 4, 30, 0xff8040);
                }
                case "gouraud_alpha128" -> {
                    routine = "gouraud";
                    Pix3D.gouraudTriangle(10, 12, 24, 12, 22, 15, 0x107f, 0x287f, 0x407f);
                    argsJson = gouraudArgs(12, 22, 15, 10, 12, 24, 0x107f, 0x287f, 0x407f);
                }
                case "textured_tex1" -> {
                    routine = "texture";
                    Pix3D.textureTriangle(6, 8, 28, 6, 26, 10, 128, 128, 128, -50, 50, -50, -50, -50, 50, 240, 240, 240, 1);
                    argsJson = textureArgs(6, 26, 10, 6, 8, 28, 128, 128, 128, -50, -50, 240, 50, -50, -50, 50, 240, 240, 1);
                }
                case "textured_shaded" -> {
                    routine = "texture";
                    Pix3D.textureTriangle(6, 8, 28, 6, 26, 10, 32, 128, 200, -50, 50, -50, -50, -50, 50, 240, 240, 240, 2);
                    argsJson = textureArgs(6, 26, 10, 6, 8, 28, 32, 128, 200, -50, -50, 240, 50, -50, -50, 50, 240, 240, 2);
                }
                case "textured_trans7" -> {
                    routine = "texture";
                    Pix3D.textureTriangle(6, 8, 28, 6, 26, 10, 32, 128, 200, -50, 50, -50, -50, -50, 50, 240, 240, 240, 7);
                    argsJson = textureArgs(6, 26, 10, 6, 8, 28, 32, 128, 200, -50, -50, 240, 50, -50, -50, 50, 240, 240, 7);
                }
                default -> throw new IllegalStateException(name);
            }

            byte[] rgba = toRGBA(px, W, H);
            writeGolden(dir, name + ".rgba", rgba);
            int nonZero = 0;
            for (int v : px) if (v != 0) nonZero++;
            if (!name.equals("gouraud_degenerate") && nonZero == 0) {
                throw new RuntimeException("triangle case " + name + " produced an empty image");
            }
            System.out.println("tri/" + name + ".rgba: " + rgba.length + " bytes, " + nonZero + " non-transparent px");

            boolean tex = routine.equals("texture");
            if (!first) manifest.append(",\n");
            first = false;
            manifest.append("  {\n")
                    .append("    \"name\": \"").append(name).append("\",\n")
                    .append("    \"routine\": \"").append(routine).append("\",\n")
                    .append("    \"width\": ").append(W).append(",\n")
                    .append("    \"height\": ").append(H).append(",\n")
                    .append("    \"prefill\": ").append(prefill).append(",\n")
                    .append("    \"lowDetail\": ").append(lowDetail).append(",\n")
                    .append("    \"hclip\": ").append(hclip).append(",\n")
                    .append("    \"trans\": ").append(trans).append(",\n")
                    .append("    \"args\": ").append(argsJson).append(",\n")
                    .append("    \"palette_dependent\": ").append(!tex).append(",\n")
                    .append("    \"texture_dependent\": ").append(tex).append(",\n")
                    .append("    \"nonZeroPixels\": ").append(nonZero).append("\n")
                    .append("  }");
        }
        // Restore lowDetail default the icon path expects to toggle itself.
        Pix3D.lowDetail = true;
        manifest.append("\n]\n");
        Files.writeString(dir.resolve("manifest.json"), manifest.toString());
        System.out.println("tri/manifest.json: " + names.length + " cases");
    }

    static String gouraudArgs(int xA, int xB, int xC, int yA, int yB, int yC, int cA, int cB, int cC) {
        return "{ \"xA\": " + xA + ", \"xB\": " + xB + ", \"xC\": " + xC
                + ", \"yA\": " + yA + ", \"yB\": " + yB + ", \"yC\": " + yC
                + ", \"colourA\": " + cA + ", \"colourB\": " + cB + ", \"colourC\": " + cC + " }";
    }

    static String flatArgs(int xA, int xB, int xC, int yA, int yB, int yC, int colour) {
        return "{ \"xA\": " + xA + ", \"xB\": " + xB + ", \"xC\": " + xC
                + ", \"yA\": " + yA + ", \"yB\": " + yB + ", \"yC\": " + yC
                + ", \"colour\": " + colour + " }";
    }

    static String textureArgs(int xA, int xB, int xC, int yA, int yB, int yC,
                              int sA, int sB, int sC, int originX, int originY, int originZ,
                              int txB, int txC, int tyB, int tyC, int tzB, int tzC, int texture) {
        return "{ \"xA\": " + xA + ", \"xB\": " + xB + ", \"xC\": " + xC
                + ", \"yA\": " + yA + ", \"yB\": " + yB + ", \"yC\": " + yC
                + ", \"shadeA\": " + sA + ", \"shadeB\": " + sB + ", \"shadeC\": " + sC
                + ", \"originX\": " + originX + ", \"originY\": " + originY + ", \"originZ\": " + originZ
                + ", \"txB\": " + txB + ", \"txC\": " + txC
                + ", \"tyB\": " + tyB + ", \"tyC\": " + tyC
                + ", \"tzB\": " + tzB + ", \"tzC\": " + tzC
                + ", \"texture\": " + texture + " }";
    }

    // ---- (c) icons + sample selection (ported from dump.ts) --------------
    static final class ObjRec {
        int id, model, recolCount;
        boolean resized;
        int ambient, contrast, certtemplate;
        boolean hasCountobj;
        int numT;
        boolean hasAlpha;
        int numFaces;
        boolean hasFlat, allGouraud;
    }

    static ObjRec readObj(int id) {
        ObjType o = ObjType.get(id);
        int modelId = o.model;
        if (Model.meta == null || modelId < 0 || modelId >= Model.meta.length) return null;
        Metadata meta = Model.meta[modelId];
        if (meta == null || meta.faceCount == 0) return null;

        boolean hasFlat = false, allGouraud = true;
        Model m = Model.load(modelId);
        if (m != null && m.faceInfo != null) {
            for (int f = 0; f < m.faceCount; f++) {
                int t = m.faceInfo[f] & 0x3;
                if (t == 1) hasFlat = true;
                if (t != 0) allGouraud = false;
            }
        }

        ObjRec r = new ObjRec();
        r.id = id;
        r.model = modelId;
        r.recolCount = o.recol_s != null ? o.recol_s.length : 0;
        r.resized = o.resizex != 128 || o.resizey != 128 || o.resizez != 128;
        r.ambient = o.ambient;
        r.contrast = o.contrast;
        r.certtemplate = o.certtemplate;
        r.hasCountobj = o.countobj != null;
        r.numT = meta.texturedFaceCount;
        r.hasAlpha = meta.faceAlphasOffset >= 0;
        r.numFaces = meta.faceCount;
        r.hasFlat = hasFlat;
        r.allGouraud = allGouraud;
        return r;
    }

    static List<String> categoriesOf(ObjRec rec) {
        List<String> t = new ArrayList<>();
        if (rec.numT > 0) t.add("textured_faces");
        if (rec.hasAlpha) t.add("alpha_faces");
        if (rec.recolCount > 0) t.add("recoloured");
        if (rec.resized) t.add("resized");
        if (rec.ambient != 0 || rec.contrast != 0) t.add("ambient_contrast");
        if (rec.certtemplate != -1) t.add("certtemplate");
        if (rec.hasCountobj) t.add("countobj_stack");
        if (rec.hasFlat) t.add("flat_faces");
        if (rec.numT == 0 && rec.allGouraud) t.add("untextured_gouraud");
        return t;
    }

    // render a plain 32x32 icon (outlineRgb 0, count 1); null if empty/absent.
    static int[] renderIcon(int id) {
        Pix32 spr;
        try {
            spr = ObjType.getSprite(0, 1, id); // Java 254: getSprite(outlineRgb, count, id)
        } catch (Exception e) {
            return null;
        }
        if (spr == null) return null;
        return spr.data;
    }

    static void dumpIcons(Path out, Map<Integer, String> nameById) throws IOException {
        Path dir = out.resolve("icons254");
        Files.createDirectories(dir);

        final int PER_CAT = 5;
        final String[] CATS = {"untextured_gouraud", "flat_faces", "textured_faces", "recoloured",
                "resized", "ambient_contrast", "certtemplate", "countobj_stack", "alpha_faces"};
        Map<String, Integer> bucketCount = new HashMap<>();
        Map<Integer, TreeSet<String>> cats = new LinkedHashMap<>();
        Map<Integer, ObjRec> chosen = new LinkedHashMap<>();

        java.util.function.BiConsumer<Integer, String> addCat = (id, c) ->
                cats.computeIfAbsent(id, k -> new TreeSet<>()).add(c);
        java.util.function.Function<String, Boolean> want = c -> bucketCount.getOrDefault(c, 0) < PER_CAT;
        java.util.function.Consumer<String> bump = c -> bucketCount.merge(c, 1, Integer::sum);

        // consider: icon must render non-empty; on success record chosen + tags.
        java.util.function.BiPredicate<ObjRec, List<String>> consider = (rec, tags) -> {
            int[] spr = renderIcon(rec.id);
            if (spr == null) return false;
            int nonZero = 0;
            for (int v : spr) if (v != 0) nonZero++;
            if (nonZero == 0) return false;
            chosen.put(rec.id, rec);
            for (String c : tags) addCat.accept(rec.id, c);
            return true;
        };

        // Seed the two mandatory items first.
        for (int id : new int[]{BRONZE_DAGGER, KNIFE}) {
            ObjRec rec = readObj(id);
            if (rec == null) throw new RuntimeException("mandatory obj " + id + " has no renderable model");
            List<String> tags = categoriesOf(rec);
            if (!consider.test(rec, tags)) throw new RuntimeException("mandatory obj " + id + " icon failed to render");
            for (String c : tags) bump.accept(c);
        }

        // Sweep all objs, filling under-full category buckets.
        int n = ObjType.count;
        for (int id = 0; id < n; id++) {
            if (chosen.containsKey(id)) continue;
            boolean allFull = true;
            for (String c : CATS) if (bucketCount.getOrDefault(c, 0) < PER_CAT) { allFull = false; break; }
            if (chosen.size() >= 44 || allFull) break;
            ObjRec rec = readObj(id);
            if (rec == null) continue;
            List<String> tags = categoriesOf(rec);
            List<String> advancing = new ArrayList<>();
            for (String c : tags) if (want.apply(c)) advancing.add(c);
            if (advancing.isEmpty()) continue;
            if (consider.test(rec, tags)) {
                for (String c : advancing) bump.accept(c);
            }
        }

        // Emit sample sorted by id + render each icon (.rgba golden + .png).
        List<Integer> ids = new ArrayList<>(chosen.keySet());
        ids.sort(Integer::compareTo);
        StringBuilder sample = new StringBuilder("[\n");
        Map<String, Integer> coverage = new LinkedHashMap<>();
        for (String c : CATS) coverage.put(c, 0);
        boolean first = true;
        for (int id : ids) {
            String debugname = nameById.getOrDefault(id, "obj_" + id);
            int[] spr = renderIcon(id);
            if (spr == null || spr.length != 32 * 32) throw new RuntimeException("icon " + id + " render failed");
            byte[] rgba = toRGBA(spr, 32, 32);
            writeGolden(dir, debugname + ".rgba", rgba);
            writePNG(dir.resolve(debugname + ".png").toFile(), spr, 32, 32);

            TreeSet<String> cset = cats.getOrDefault(id, new TreeSet<>());
            for (String c : cset) coverage.merge(c, 1, Integer::sum);
            if (!first) sample.append(",\n");
            first = false;
            StringBuilder catArr = new StringBuilder("[");
            boolean cf = true;
            for (String c : cset) { if (!cf) catArr.append(", "); cf = false; catArr.append('"').append(c).append('"'); }
            catArr.append("]");
            sample.append("  {\n")
                    .append("    \"id\": ").append(id).append(",\n")
                    .append("    \"debugname\": \"").append(debugname).append("\",\n")
                    .append("    \"categories\": ").append(catArr).append("\n")
                    .append("  }");
        }
        sample.append("\n]\n");
        Files.writeString(dir.resolve("sample.json"), sample.toString());

        System.out.println("icons254: " + ids.size() + " icons written (.rgba + .png)");
        System.out.println("category coverage: " + coverage);
        List<String> missing = new ArrayList<>();
        for (String c : CATS) if (coverage.getOrDefault(c, 0) == 0) missing.add(c);
        if (!missing.isEmpty()) System.out.println("NOTE: categories with zero coverage at 254: " + missing);
    }

    private IconDump() { }
}
