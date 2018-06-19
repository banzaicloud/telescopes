export class Products {
  // provider: string;
  Products: Product[];
}

export class Product {
  type: string;
  cpusPerVm: number;
  memPerVm: number;
  onDemandPrice: number;
  spotPrice: SpotPrice[];
  ntwPerf: string;
}

export class SpotPrice {
  zone: string;
  price: string;
}

export class DisplayedProduct {
  constructor(private type: string,
              private cpus: string,
              private mem: string,
              private regularPrice: string,
              private spotPrice: string,
              private ntwPerf: string) {
  }
}

export class Region {
  id: string;
  name: string;
}


