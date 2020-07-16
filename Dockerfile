FROM deepfabric/vectodb as builder

COPY . /root/go/src/github.com/deepfabric/beevector
WORKDIR /root/go/src/github.com/deepfabric/beevector

RUN source scl_source enable devtoolset-8 \
    && make beevector

FROM deepfabric/vectodb-runtime
COPY --from=builder /root/go/src/github.com/deepfabric/beevector/dist/beevector /usr/local/bin/beevector
ENTRYPOINT ["/usr/local/bin/beevector"]