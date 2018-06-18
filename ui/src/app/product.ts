export class Products {
  provider: string;
  products: Product[];
}

export class Product {
  type: string;
  cpusPerVm: number;
  memPerVm: number;
  onDemandPrice: number;
}

export class DisplayedProduct {
  constructor(private type: string,
              private cpus: string,
              private mem: string,
              private regularPrice: string) {
  }
}

export class Region {
  id: string;
  name: string;
}


