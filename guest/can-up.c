#include <errno.h>
#include <linux/can/netlink.h>
#include <linux/if_link.h>
#include <linux/netlink.h>
#include <linux/rtnetlink.h>
#include <net/if.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/socket.h>
#include <unistd.h>

static int add_attr(struct nlmsghdr *message, size_t capacity, int type,
                    const void *data, size_t data_length) {
    size_t length = RTA_LENGTH(data_length);
    size_t offset = NLMSG_ALIGN(message->nlmsg_len);
    if (offset + RTA_ALIGN(length) > capacity) {
        return -1;
    }

    struct rtattr *attribute = (struct rtattr *)((char *)message + offset);
    attribute->rta_type = type;
    attribute->rta_len = length;
    if (data_length > 0) {
        memcpy(RTA_DATA(attribute), data, data_length);
    }
    message->nlmsg_len = offset + RTA_ALIGN(length);
    return 0;
}

static struct rtattr *begin_nested(struct nlmsghdr *message, size_t capacity,
                                   int type) {
    size_t offset = NLMSG_ALIGN(message->nlmsg_len);
    if (add_attr(message, capacity, type, NULL, 0) < 0) {
        return NULL;
    }
    return (struct rtattr *)((char *)message + offset);
}

static void end_nested(struct nlmsghdr *message, struct rtattr *attribute) {
    attribute->rta_len = (char *)message + message->nlmsg_len - (char *)attribute;
}

int main(int argc, char **argv) {
    if (argc != 3) {
        fprintf(stderr, "usage: can-up INTERFACE BITRATE\n");
        return 2;
    }

    unsigned int interface_index = if_nametoindex(argv[1]);
    if (interface_index == 0) {
        perror("can-up: if_nametoindex");
        return 2;
    }

    char *end = NULL;
    unsigned long bitrate = strtoul(argv[2], &end, 10);
    if (end == argv[2] || *end != '\0' || bitrate == 0 || bitrate > UINT32_MAX) {
        fprintf(stderr, "can-up: invalid bitrate %s\n", argv[2]);
        return 2;
    }

    int fd = socket(AF_NETLINK, SOCK_RAW, NETLINK_ROUTE);
    if (fd < 0) {
        perror("can-up: socket");
        return 2;
    }

    struct sockaddr_nl address = {.nl_family = AF_NETLINK};
    if (bind(fd, (struct sockaddr *)&address, sizeof(address)) < 0) {
        perror("can-up: bind");
        close(fd);
        return 2;
    }

    struct {
        struct nlmsghdr header;
        struct ifinfomsg interface;
        char attributes[512];
    } request;
    memset(&request, 0, sizeof(request));
    request.header.nlmsg_len = NLMSG_LENGTH(sizeof(struct ifinfomsg));
    request.header.nlmsg_type = RTM_NEWLINK;
    request.header.nlmsg_flags = NLM_F_REQUEST | NLM_F_ACK;
    request.header.nlmsg_seq = 1;
    request.interface.ifi_family = AF_UNSPEC;
    request.interface.ifi_index = (int)interface_index;
    request.interface.ifi_flags = IFF_UP;
    request.interface.ifi_change = IFF_UP;

    struct rtattr *link_info = begin_nested(&request.header, sizeof(request), IFLA_LINKINFO);
    if (link_info == NULL ||
        add_attr(&request.header, sizeof(request), IFLA_INFO_KIND, "can", 4) < 0) {
        fprintf(stderr, "can-up: netlink message is too large\n");
        close(fd);
        return 2;
    }

    struct rtattr *info_data = begin_nested(&request.header, sizeof(request), IFLA_INFO_DATA);
    struct can_bittiming timing = {.bitrate = (uint32_t)bitrate};
    if (info_data == NULL ||
        add_attr(&request.header, sizeof(request), IFLA_CAN_BITTIMING,
                 &timing, sizeof(timing)) < 0) {
        fprintf(stderr, "can-up: netlink message is too large\n");
        close(fd);
        return 2;
    }
    end_nested(&request.header, info_data);
    end_nested(&request.header, link_info);

    if (send(fd, &request, request.header.nlmsg_len, 0) < 0) {
        perror("can-up: send");
        close(fd);
        return 2;
    }

    char response[4096];
    ssize_t received = recv(fd, response, sizeof(response), 0);
    if (received < 0) {
        perror("can-up: receive acknowledgement");
        close(fd);
        return 2;
    }
    close(fd);

    for (struct nlmsghdr *header = (struct nlmsghdr *)response;
         NLMSG_OK(header, received); header = NLMSG_NEXT(header, received)) {
        if (header->nlmsg_type == NLMSG_ERROR) {
            struct nlmsgerr *error = NLMSG_DATA(header);
            if (error->error != 0) {
                errno = -error->error;
                perror("can-up: RTM_NEWLINK");
                return 2;
            }
            printf("CAN_READY interface=%s bitrate=%lu\n", argv[1], bitrate);
            return 0;
        }
    }

    fprintf(stderr, "can-up: no netlink acknowledgement received\n");
    return 2;
}
