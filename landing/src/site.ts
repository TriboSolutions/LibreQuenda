export const REPO = "https://github.com/TriboSolutions/LibreQuenda";
export const REPO_API = "https://api.github.com/repos/TriboSolutions/LibreQuenda";
export const INSTALL = `mkdir librequenda && cd librequenda
curl -fsSLO https://raw.githubusercontent.com/TriboSolutions/LibreQuenda/main/install.sh
bash install.sh`;

export const SITE = "https://librequenda.app";
/** A LibreQuenda with demo data, to try it */
export const DEMO = "https://demo.librequenda.app";
/**
 * Tribo Solutions, who makes LibreQuenda: the same data and the same schema.org @id as on
 * tribosolutions.es, so search engines see one organization across both sites.
 */
export const ORG = {
  name: "Tribo Solutions",
  legalName: "Tribo Solutions AMX, S.L.",
  taxId: "B23940000",
  url: "https://tribosolutions.es",
  id: "https://tribosolutions.es/#organization",
  logo: "https://tribosolutions.es/logos/logo_tribo_blue.png",
  email: "contact@tribosolutions.es",
  phone: "+34 633 885 601",
  address: { street: "Calle Arenaza, 43", locality: "Oleiros", region: "A Coruña, Galicia", postalCode: "15171", country: "ES" },
  github: "https://github.com/TriboSolutions",
  linkedin: "https://www.linkedin.com/company/tribo-solutions/",
};
/** Search engine site verification (Google Search Console, Bing Webmaster Tools) */
export const VERIFY = { google: "-ZllA27lA55OYlwz_5Pw6Pduh502l5478lgEokaftxg", bing: "8BAA64E7B71B62C11F6FDEEE9F88C7AC" };
export const LICENSE_URL = "https://www.gnu.org/licenses/agpl-3.0.html";
/** The social preview image, 1200×630, in public/ */
export const OG_IMAGE = { path: "/og.jpg", width: 1200, height: 630 };
/** The demo app on Google Play; on Android it opens the Play Store app */
export const PLAY = "https://play.google.com/store/apps/details?id=gal.tribo.librequenda";
