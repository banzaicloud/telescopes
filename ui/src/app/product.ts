export class Products {
  provider: string;
  products: Product[];
}

export class Product {
  type: string;
  cpusPerVm: number;
  memPerVm: number;
  onDemandPrice: number;
  spotPrice: SpotPrice[];
}

export class SpotPrice {
  zone: string;
  price: number;
}

export class DisplayedProduct {
  constructor(private type: string,
              private cpus: string,
              private mem: string,
              private regularPrice: string,
              private spotPrice: string) {
  }
}

export class Region {
  id: string;
  name: string;
}


