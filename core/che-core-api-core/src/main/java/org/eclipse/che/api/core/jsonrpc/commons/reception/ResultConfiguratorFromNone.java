/*******************************************************************************
 * Copyright (c) 2012-2017 Codenvy, S.A.
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v1.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v10.html
 *
 * Contributors:
 *   Codenvy, S.A. - initial API and implementation
 *******************************************************************************/
package org.eclipse.che.api.core.jsonrpc.commons.reception;

import org.eclipse.che.api.core.jsonrpc.commons.JsonRpcErrorTransmitter;
import org.eclipse.che.api.core.jsonrpc.commons.RequestHandlerManager;
import org.slf4j.Logger;

import static com.google.common.base.Preconditions.checkNotNull;
import static org.slf4j.LoggerFactory.getLogger;

/**
 * Result configurator provide means to configure result type in a
 * response that is to be received. Result types that are supported:
 * {@link String}, {@link Boolean}, {@link Double}, {@link Void} and
 * DTO. This configurator is used when we have no defined request params.
 */
public class ResultConfiguratorFromNone {
    private final static Logger LOGGER = getLogger(ResultConfiguratorFromNone.class);

    private final RequestHandlerManager requestHandlerManager;

    private final String method;

    ResultConfiguratorFromNone(RequestHandlerManager requestHandlerManager, String method) {
        this.requestHandlerManager = requestHandlerManager;

        this.method = method;
    }

    public <R> FunctionConfiguratorNoneToMany<R> resultAsListOfDto(Class<R> rClass) {
        checkNotNull(rClass, "Result class must not be null");
        LOGGER.debug("Configuring incoming request result: method: " + method + ", result list items class: " + rClass);

        return new FunctionConfiguratorNoneToMany<>(requestHandlerManager, method, rClass);
    }

    public FunctionConfiguratorNoneToMany<String> resultAsListOfString() {
        LOGGER.debug("Configuring incoming request result: method: " + method + ", result list items class: " + String.class);

        return new FunctionConfiguratorNoneToMany<>(requestHandlerManager, method, String.class);
    }

    public FunctionConfiguratorNoneToMany<Double> resultAsListOfDouble() {
        LOGGER.debug("Configuring incoming request result: method: " + method + ", result list items class: " + Double.class);

        return new FunctionConfiguratorNoneToMany<>(requestHandlerManager, method, Double.class);
    }

    public FunctionConfiguratorNoneToMany<Boolean> resultAsListOfBoolean() {
        LOGGER.debug("Configuring incoming request result: method: " + method + ", result list items class: " + Boolean.class);

        return new FunctionConfiguratorNoneToMany<>(requestHandlerManager, method, Boolean.class);
    }

    public <R> FunctionConfiguratorNoneToOne<R> resultAsDto(Class<R> rClass) {
        checkNotNull(rClass, "Result class must not be null");
        LOGGER.debug("Configuring incoming request result: method: " + method + ", result object class: " + rClass);

        return new FunctionConfiguratorNoneToOne<>(requestHandlerManager, method, rClass);
    }

    public FunctionConfiguratorNoneToOne<String> resultAsString() {
        LOGGER.debug("Configuring incoming request result: method: " + method + ", result object class: " + String.class);

        return new FunctionConfiguratorNoneToOne<>(requestHandlerManager, method, String.class);
    }

    public FunctionConfiguratorNoneToOne<Double> resultAsDouble() {
        LOGGER.debug("Configuring incoming request result: method: " + method + ", result object class: " + Double.class);

        return new FunctionConfiguratorNoneToOne<>(requestHandlerManager, method, Double.class);
    }

    public FunctionConfiguratorNoneToOne<Boolean> resultAsBoolean() {
        LOGGER.debug("Configuring incoming request result: method: " + method + ", result object class: " + Boolean.class);

        return new FunctionConfiguratorNoneToOne<>(requestHandlerManager, method, Boolean.class);
    }

    public ConsumerConfiguratorNoneToNone noResult() {
        LOGGER.debug("Configuring incoming request result: " + "method: " + method + ", no result is expected.");

        return new ConsumerConfiguratorNoneToNone(requestHandlerManager, method);
    }
}
