/*
 * Copyright (c) 2015-2017 Codenvy, S.A.
 * All rights reserved. This program and the accompanying materials
 * are made available under the terms of the Eclipse Public License v1.0
 * which accompanies this distribution, and is available at
 * http://www.eclipse.org/legal/epl-v10.html
 *
 * Contributors:
 *   Codenvy, S.A. - initial API and implementation
 */
'use strict';
import {ProjectSourceSelectorService} from './project-source-selector.service';
import {ProjectSource} from './project-source.enum';

/**
 * This class is handling the controller for the project selector.
 *
 * @author Oleksii Kurinnyi
 * @author Oleksii Orel
 */
export class ProjectSourceSelectorController {
  /**
   * Updates widget function.
   */
  updateWidget: Function;

  /**
   * Project selector service.
   */
  private projectSourceSelectorService: ProjectSourceSelectorService;
  /**
   * Available project sources.
   */
  private projectSource: Object;
  /**
   * Selected project's source.
   */
  private selectedSource: ProjectSource;
  /**
   * button's values by Id.
   */
  private buttonValues: { [butonId: string]: boolean } = {};
  /**
   * Selected button's Id.
   */
  private selectButtonId: string;
  /**
   * Selected template.
   */
  private projectTemplate: che.IProjectTemplate | {};

  /**
   * Default constructor that is using resource injection
   * @ngInject for Dependency injection
   */
  constructor(projectSourceSelectorService: ProjectSourceSelectorService) {
    this.projectSourceSelectorService = projectSourceSelectorService;

    this.projectSource = ProjectSource;
    this.selectedSource = ProjectSource.SAMPLES;
  }

  /**
   * Returns list of project templates which are ready to be imported.
   *
   * @return {Array<che.IProjectTemplate>}
   */
  getProjectTemplates(): Array<che.IProjectTemplate> {
    return this.projectSourceSelectorService.getProjectTemplates();
  }

  /**
   * Add project template from selected source to the list.
   */
  addProjectTemplate(): void {
    this.projectSourceSelectorService.addProjectTemplateFromSource(this.selectedSource);
  }

  /**
   * Updates widget data.
   * @param state {boolean}
   * @param id {string}
   * @param template {che.IProjectTemplate}
   */
  updateData({state, id, template}: { state: boolean, id: string, template?: che.IProjectTemplate }): void {
    if (!state) {
      return;
    }
    if (!template) {
      this.selectButtonId = id;
      this.projectTemplate = {};
    } else {
      this.selectButtonId = 'projectTemplate';
      this.projectTemplate = template;
    }
    // leave only one selected button
    const keys = Object.keys(this.buttonValues);
    keys.forEach((key: string) => {
      if (key !== id && this.buttonValues[key]) {
        this.buttonValues[key] = false;
      }
    });
    if (angular.isFunction(this.updateWidget)) {
      // update widget
      this.updateWidget();
    }
  }
}
