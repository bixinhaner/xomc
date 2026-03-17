<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
    #addOrEditConfigPage .split-line { border: none; border-top: 1px solid #e9e9e9; margin: 10px 0 20px 0; }
    #addOrEditConfigPage .el-select-mini input { min-height: 26px; }
    #addOrEditConfigPage .basicInfoBox { margin: 16px 40px 10px; }
	#addOrEditConfigPage .originalBox .el-checkbox { padding: 3px 6px 0 10px; }
	#addOrEditConfigPage .addVersionWarp { width: 400px; height: 254px; border-radius: 4px; text-align: center; border: 1px solid #E9EDF9;}
	#addOrEditConfigPage .addVersionBtn { font-size: 20px; padding: 100px 0 10px; }
	#addOrEditConfigPage .selectVersionBox .el-form-item__content { margin-left: 0px !important; }
	#addOrEditConfigPage .selectVersionBox .el-form-item__content .el-input { width: 300px; }
	#addOrEditConfigPage .selectMethodBox { display: flex; flex-direction: row; justify-content: space-between; width: 80%; }
	#addOrEditConfigPage .selectMethodBox .el-radio-button__inner { display: flex; border: 1px solid #E9EDF9; min-width: 240px; height: 50px; background: #F5F7FE; line-height: 50px; border-radius: 4px; padding: 0; font-size: unset; font-weight: normal;text-align: center;}
	#addOrEditConfigPage .selectMethodBox .el-radio-button__inner [class*=el-icon-]+span { margin-left: 0; }
	#addOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .methodTitle {color:var(--main-color);}
	#addOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .commonIconStyle,#addOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover,#addOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover .commonIconStyle,#addOrEditConfigPage .el-radio:hover, #addOrEditConfigPage .el-radio .el-radio__inner:hover { color:var(--main-color); background: rgba(var(--main-color-rgba1),0.1); border-color:var(--main-color) !important; }
	#addOrEditConfigPage .licenseBox .el-query{ right: 0; }
	#addOrEditConfigPage .licenseImportBox, .specifyModifyRightBox .licenseImportBox { margin: 0 10px; width: 26px; height: 26px; border: 1px solid #D7D7E6; text-align: center; border-radius: 8px; }
	#addOrEditConfigPage .licenseImportBox i,.specifyModifyRightBox .licenseImportBox i  { font-size: 12px; line-height: 26px; }
	#addOrEditConfigPage .leftWarp {flex: 1; height: 100%; position: relative; overflow: hidden; flex-direction: column; border-radius: 10px; border: 1px solid #E9EDF9;  background: #FFFFFF;}
	#addOrEditConfigPage .titleBox { width: 100%; height: 38px; background: #FFFFFF; line-height: 38px; position: absolute; top: 0; left: 0; z-index: 100; }
	#addOrEditConfigPage .footerBox { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	#addOrEditConfigPage .AddTitle { padding: 0 30px; }
	#addOrEditConfigPage .closeIconBox .el-icon { font-size: 14px !important; color: #7a7992; margin: 12px 16px;}
	#addOrEditConfigPage .specifyModifyRightBox { flex: 0 580px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; border: 1px solid #E9EDF9;background: #FFFFFF;}
	#addOrEditConfigPage .rightBox320 { flex: 0 320px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; border: 1px solid #E9EDF9;background: #FFFFFF;}
	#addOrEditConfigPage .rightBox { flex: 0 1 360px; height: 100%;  margin-left: 10px; border-radius: 10px; overflow: hidden; border: 1px solid #E9EDF9;background: #FFFFFF;}
	#addOrEditConfigPage .mainContent { width: 100%; overflow-y: scroll; height: 100%; position: absolute; top: 40px; z-index: 0; box-sizing: border-box; }
	#addOrEditConfigPage .commonTitle { height: 38px; line-height: 38px; }
	#addOrEditConfigPage .rightHeaderBox .addTitle { padding: 0; }
	#addOrEditConfigPage .rightTextBox { padding: 10px 0; }
	#addOrEditConfigPage .commonBorderBottom { border-bottom: 1px solid #E9EDF9; }
	#addOrEditConfigPage .commonBorderTop { border-top: 1px solid #E9EDF9; }
	#addOrEditConfigPage .originalVersionBox { padding: 16px 0 0; }
	#addOrEditConfigPage .originalVersionBox .el-input{ width: 310px }
	#addOrEditConfigPage .width280 .el-input{ width: 280px; }
	#addOrEditConfigPage .originalVersionBox .el-input__inner{ height: 30px; line-height: 30px; }
	#addOrEditConfigPage .originalVersionBox .el-input-group__append { padding: 0 8px; background-color: #FFFFFF; }
	#addOrEditConfigPage .originalVersionBox .el-input-group__append .el-icon { font-size: 14px; }
	#addOrEditConfigPage .originalVersionBox .el-input-group__append .el-icon:before { color: #7A7992; }
	#addOrEditConfigPage .originalVersionBox .el-table tr { display: none; }
	#addOrEditConfigPage .ipErrorTip { color: #FA5555; font-size: 12px; }
	#addOrEditConfigPage .versionResultBox { border-radius: 4px; background-color: #FFFFFF; margin-top: 5px; width: 318px; max-height: 122px; padding: 5px 0; overflow: auto; border: 1px solid #E9EDF9;}
	#addOrEditConfigPage .versionResultBox .el-form-item { margin-right: 0 !important;}
	#addOrEditConfigPage .form-suffix { position: relative; margin: 3px 0 0 20px; padding: 0; width:278px; border: none; background: #fff; }
	#addOrEditConfigPage .form-suffix:hover { background: #F4F9FF; border-radius: 100px; }
	#addOrEditConfigPage .form-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 0px; color: var(--main-color); font-size: 14px;}
	#addOrEditConfigPage .form-suffix:hover .deleteVersion { display: inline-block !important; }
	#addOrEditConfigPage .form-suffix .text { padding-left: 10px; color: #666666; }
	#addOrEditConfigPage .selectedVersion .queryGroup .el-input__inner, #addOrEditConfigPage .curImportQuery .el-input__inner { width: 220px !important; }
	#addOrEditConfigPage .selectedVersion .el-query .advanceQuery { height: 28px; }
	#addOrEditConfigPage .selectedVersion .el-query .advanceQuery .el-input.el-input--small { width: 230px !important; }
	#addOrEditConfigPage .selectedVersion .el-query { right: 0px; }
	#addOrEditConfigPage .operBtn { width: 26px; height: 26px; border: 1px solid #D7D7E6; border-radius: 8px; margin-top: 1px; }
	#addOrEditConfigPage .operBtn .el-icon { line-height: 26px; }
	#addOrEditConfigPage .operBtn .el-icon:before, #addOrEditConfigPage .importIcon:before { color: #7A7992; }
	.container .group { padding-left: 26px; }
	#addOrEditConfigPage .mainContent .el-form-item { margin-bottom: 22px; }
	#addOrEditConfigPage .mainContent .el-form-item__label { line-height: 28px; }
	#addOrEditConfigPage .el-radio.is-bordered { max-width: 160px; height: 30px; padding: 7px 12px; }
	#addOrEditConfigPage .el-radio-group .el-radio__label { font-size: 12px; }
	#addOrEditConfigPage .enableCommon .el-switch { margin-top: 4px; }
	#addOrEditConfigPage .selectCommon .el-select .el-input { width: 150px; }
	#addOrEditConfigPage .inputCommon .el-input__inner { border-radius: 4px; }
    #addOrEditConfigPage .paramTwoBox { padding: 10px 20px; }
    #addOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__arrow { position: absolute; left: 0; top: 0px; }
    #addOrEditConfigPage .infoSpecifiedDevice .el-collapse-item {  position: relative; }
    #addOrEditConfigPage .infoSpecifiedDevice .el-icon-arrow-right {  font-size: 16px; }
    #addOrEditConfigPage .infoSpecifiedDevice .el-icon-arrow-right:before { content: "\e639"; color: #BBB;  }
    #addOrEditConfigPage .infoSpecifiedDevice .is-active.el-icon-arrow-right:before { content: "\e638"; color: #BBB; }
    #addOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__arrow.is-active { transform: rotate(0deg); }
    #addOrEditConfigPage .infoSpecifiedDevice .el-collapse { border-top: 1px solid #fff; border-bottom: 1px solid #fff; }
    #addOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__wrap { border-bottom: 1px solid #fff; }
    #addOrEditConfigPage .infoSpecifiedDevice .btn-next .el-icon-arrow-right:before { content: "\e794"; }
	#addOrEditConfigPage .commonText { padding: 3px 0; }
	#addOrEditConfigPage .commonText1 { padding: 0 10px; }
	#addOrEditConfigPage .el-collapse-item__content { padding-bottom: 0; }
	#addOrEditConfigPage .enbIdItem .el-textarea__inner { height: 160px; resize: none; border-radius: 4px; }
	#addOrEditConfigPage .reginDeployingBox .el-tabs--border-card { border: 1px solid #E9EDF9; box-shadow: none; -webkit-box-shadow: none; }
	#addOrEditConfigPage .reginDeployingBox .el-tabs .el-tabs__header,
	#addOrEditConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header { border-bottom: 1px solid #E9EDF9; }
	#addOrEditConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header { background-color: #FFFFFF; }
	#addOrEditConfigPage .reginDeployingBox .el-tabs__item { font-size: 14px; color: #7A7992; font-weight: normal;  }
	#addOrEditConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header .el-tabs__item.is-active { color: #7A7992; font-weight: bold;  border-right-color: #E9EDF9; border-left-color: #E9EDF9; }
	#addOrEditConfigPage .reginDeployingBox .el-tabs--border-card>.el-tabs__header .el-tabs__item:not(.is-disabled):hover { color: #4D84FF; }
    #addOrEditConfigPage .modifyForm .el-form-item { margin-bottom: 30px; display: inline-block; margin-right: 40px; }
	#addOrEditConfigPage .modifyForm .el-form-item .el-form-item__label { margin-bottom: 4px; }
    #addOrEditConfigPage .table-suffix { border: none; position: relative; }
	#addOrEditConfigPage .table-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 2px; color : var(--main-color); font-size: 14px;}
	#addOrEditConfigPage .table-suffix:hover .deleteVersion { display: inline-block !important; }
	#addOrEditConfigPage .table-suffix .text { padding-left: 10px; color: #666666; }
	#addOrEditConfigPage .disabledClass { cursor: not-allowed !important; opacity: 0.4; }
	#addOrEditConfigPage .disabledClass:before {color: #c0c4cc;  cursor: not-allowed !important; }
	#addOrEditConfigPage .defaultClass { cursor: pointer; }
	#addOrEditConfigPage .paramPoolWarp .el-form-item { width: 49%; display: inline-block; flex-direction: row; }
	#addOrEditConfigPage .paramPoolWarp .el-form-item .el-input { width: 186px; }
	#addOrEditConfigPage .commonLeft10 { margin-left: 2px; }
	#addOrEditConfigPage .fielset-cls {margin-top: 15px; padding: 0px 20px; border: none; border-top: 1px solid #e9e9e9; height: 0px; overflow: hidden;}
	#addOrEditConfigPage .fielset-cls.extended {height: auto; padding: 10px 20px; border-radius: 5px; border: 1px solid #e9e9e9;}
	#addOrEditConfigPage .fielset-cls .el-form-item { max-width: 360px;}
	#addOrEditConfigPage .basicLeftLabel { display: flex; }
	#addOrEditConfigPage .basicLeftLabel .el-form-item__label { width: 125px; }
	#addOrEditConfigPage .commonFormFotter { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	#addOrEditConfigPage .el-form-item__error { padding-top: 0 !important; }
	.dialogIpsec { margin: 50px auto 0 !important;}
	#addOrEditConfigPage .el-input.is-disabled .el-input__inner { height: 26px !important; }
	.container .slide-position-top .el-icon-close { font-size: 16px; }
	#addOrEditConfigPage .marginRight10 { margin-right: 10px; }
	#addOrEditConfigPage .contentHeight { height: calc(100% - 100px); overflow: scroll; }
	#addOrEditConfigPage .selectSnBtn{ border:none; }
	#addOrEditConfigPage .selectSnBtn a{ text-decoration:none; font-size:12px; color:#4D84FF; border-bottom:1px solid #4D84FF; }
	#addOrEditConfigPage .selectSnBtn:hover,#addOrEditConfigPage .selectSnBtn:focus,#addOrEditConfigPage .selectSnBtn:active,#addOrEditConfigPage .selectSnBtn:visited{ background:#FFFFFF; }	
	#addOrEditConfigPage .newIconBoxCls-bt .el-icon-circle-close:before { content: '\e778'; font-size: 12px !important; }
	#addOrEditConfigPage .el-icon-menu-system:before { color: #7A7992; }
	#addOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .el-icon-menu-system:before,#addOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover .el-icon-menu-system:before { color:var(--main-color); }
	#addOrEditConfigPage .validateItem .el-input-group__append{ border:none; background:none; padding: 0px 10px; }
	#addOrEditConfigPage .is-error .el-input-group__append{ color:#FA5555; }
	#addOrEditConfigPage .validateItem .el-input__inner{ width:186px; }
	#addOrEditConfigPage .validateItem .el-input-group__append, #addOrEditConfigPage .validateRightItem .el-input-group__append{ border:none; background:none; }
	#addOrEditConfigPage .validateItem .el-form-item__error,#addOrEditConfigPage .validateRightItem .el-form-item__error{ display:none; }
	#addOrEditConfigPage .validateRightItem .el-input-group__append{ border:none; background:none; padding: 0; }
	#addOrEditConfigPage .validateRightItem .el-input__inner{ width:280px; }
	#addOrEditConfigPage .validateRightItem .el-input{ display: block; }
	#addOrEditConfigPage .specialItemCls{ border: 1px solid #DFE2EE; position: relative; margin: 20px; padding: 10px 20px 0; border-radius: 8px; }
	#addOrEditConfigPage .specialItemDelIcon{ height: 26px; width: 26px; display: flex; align-items: center; justify-content: center; border: 1px solid #DFE2EE; background-color: #FFFFFF; border-radius: 100px; position: absolute; right:-12px; top:-12px; z-index: 231 }
	.specialItemCls .validate-item { width: 20%; }
	.specialItemCls .validate-item .el-input__inner{ width:200px; }
	.specialItemCls .validate-item .el-input-group__append{ border:none; background:none; padding: 0px 10px; }
	.specialItemCls .validate-item .el-form-item__error{display:none;}
	#addOrEditConfigPage .paramsItemWarp .el-form-item { width: 49%; display:inline-block; flex-direction:row; }
	#addOrEditConfigPage .paramsBasicItemWarp .el-form-item { width: 40%; display:inline-block; flex-direction:row; }
	#addOrEditConfigPage .el-tabs--top .el-tabs__item.is-top:last-child { border-left: 1px solid #E9EDF9; }
	#addOrEditConfigPage .plmnWarp .el-input__inner { width: 200px !important; }
	#addOrEditConfigPage .plmnWarp .el-input-group__append {  border: 0; background: #FFFFFF; }
	#addOrEditConfigPage .tieleTipCircle { width: 8px; margin: 14px 10px 0 0; height: 8px; border-radius: 50%; background: rgba(0, 0, 0, 0.8); display: inline-block; }
</style>
<div id="addOrEditConfigPage" style='background: #F6F7FB; position: relative;'>
   <div class='commonFlex commonContent' style='height: 99.8%; width:100%; '>
   		<!-- 左侧主体内容 -->
   		<div class='leftWarp'>
   			<div class='commonFlex commonContent titleBox commonTitle commonBorderBottom'>
   				<span class='AddTitle commonTextWeight14'>{{policyTitle}}</span>
   				<div class="newIconBoxCls-bt" style="right:15px;top:7px;" @click="enbAddOrUpdateCancel">
					<span class="el-icon el-icon-circle-close"></span>
				</div>
   			</div>
   			<div class='mainContent' style='bottom: 70px; height: auto;'>
   				<el-form ref="addOrEditForm" :model="addOrEditForm" :rules="addOrEditRule" label-position="top" label-width="125">			
                    <div class="basicInfoBox">
			        	<div class="group-title not-extend">
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("JiBenXinXi") %></span>
						</div>
						<div style='padding-left: 26px; padding-top: 16px;'>
							<el-form-item label='<%=rb.getString("SheZhiKaiGuan") %>' prop='selfStartEnable' class='enableCommon basicLeftLabel'>
								<el-switch v-model="addOrEditForm.selfStartEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="isReadOnly"></el-switch>
							</el-form-item>
							<el-form-item label="<%=rb.getString("CeLueMingChen") %>" prop="policyName" class='inputCommon basicLeftLabel'>
				                <el-input v-model="addOrEditForm.policyName" :disabled="isReadOnly" style="width: 300px;"></el-input>
				            </el-form-item>
				            <el-form-item label="<%=rb.getString("ChanPinLeiXingBiaoZhi") %>" prop="productType" placeholder="<%=rb.getString("QingXuanZe") %>" class='selectCommon basicLeftLabel'>
				                <el-select v-model="addOrEditForm.productType" :disabled="isReadOnly || curType=='modify'" @change="productTypeChange">
				                    <el-option v-for="item in productList" :label="item.name" :value="item.value"></el-option>
				                </el-select>
				            </el-form-item>
				            <el-form-item label="<%=rb.getString("ZhiXingFangShi") %>" prop="executeType" style='margin-bottom: 30px;' class='basicLeftLabel'>
				                <el-radio-group v-model="addOrEditForm.executeType" :disabled="isReadOnly">
				                    <el-radio label="0" border><%=rb.getString("eNBZiDongZhiXing") %></el-radio>
				                    <el-radio label="1" border style='margin-left: 20px;'><%=rb.getString("eNBShouDongZhiXing") %></el-radio>
				                </el-radio-group>
				            </el-form-item>
						</div>
					</div>
					<hr class="split-line">
					<div class='basicInfoBox'><!-- Software Upgrade,license model-->
			        	<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 16px;'><%=rb.getString("KeYiXuanZeYiXiaMoKuaiJinXingPeiZhi") %></span>
			        	<div style="padding-bottom: 6px;">
			        		<el-radio-group v-model='functionModulesSelect' class='selectMethodBox' @change='selectMethodChange'>
								<el-radio-button label="0" class='moduleConfigBox'>
									<i class='el-icon el-icon-status-upgrading commonIconStyle'></i>
			        				<span class='commonTextNormal14  methodTitle'><%=rb.getString("RuanJianShengJi") %></span>
								</el-radio-button>
								<el-radio-button label="1" class='moduleConfigBox' v-if="curProductName != 'DXDF'">
									<i class='el-icon el-icon-operation-edit commonIconStyle'></i>
			        				<span class='commonTextNormal14  methodTitle'>License</span>
								</el-radio-button>	
								<el-radio-button label="2" class='moduleConfigBox'>
									<i class='el-icon el-icon-menu-system commonIconStyle'></i>
			        				<span class='commonTextNormal14  methodTitle'><%=rb.getString("ENBCanShuZiPeiZhi") %></span>
			        			</el-radio-button>							
							</el-radio-group>
			        	</div>
			        </div>
			        <div v-show="functionModulesSelect == '0'" class="basicInfoBox commonColor" style='margin-top: 20px;'>	<!-- Software Upgrade -->
			            <div class="group-title not-extend">
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("RuanJianShengJi") %></span>
							<el-switch v-model="addOrEditForm.upgradeEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="isReadOnly"></el-switch>
						</div>
						<div style='padding-left: 26px;'>
				            <span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 20px; margin-top: 14px;'><%=rb.getString("NinKeYiShouDongHuoLieBiaoXuanZeChuShiBanBenHao") %></span>
				            <div class='commonFlex originalBox'>
				            	<span class='commonSize14'><%=rb.getString("ChuShiBanBen") %></span>
				                <el-checkbox v-model="addOrEditForm.specifyVersionType" true-label="1" false-label="0" :disabled="isReadOnly"></el-checkbox>
				            	<span class='commonSize12ExportText' style='margin-top: 1px;'><%=rb.getString("SuoYouAny") %></span>
				            </div>  
							<div class='commonFlex' style='padding: 10px 0 4px;'>
								<div class='addVersionWarp' v-show='addVersionBtnShow'>
									<div v-if="addOrEditForm.specifyVersionType == '1' || isReadOnly == true">
										<i class='el-icon el-icon-plus addVersionBtn commonDisplayBlock disabledClass'></i>
									</div>
									<div v-else>
										<i @click='addVersionClick' class='el-icon el-icon-plus addVersionBtn commonDisplayBlock defaultClass'></i>
									</div>
									<span class='commonTitle12'><%=rb.getString("NinKeYiTianJiaYuanShiBanBen") %></span>
								</div> 
								<div class='addVersionWarp ' v-show='resultOriginalVersionShow'>
									<el-ctable ref="resultOriginalVersionTable" row-key="originalVersion" class='commonBorderRadius' style='margin-bottom: 10px;'
										:data="resultOriginalVersionList.filter(item=>{
											return item.originalVersion.indexOf(searchValue) > -1
										})" :pagination="false" :rownumber="false">
										<div slot="toolbar">
											<div class='commonFlex commonToolBarBox commonContent selectedVersion'>
												<div class='queryGroup' style='margin-right: 6px;'> 
													<el-input v-model="searchValue" class='pairgrid-query' placeholder='<%=rb.getString("ChuShiBanBen")%>' style='width: 220px;'></el-input>
													<i class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
												</div>
												<div v-show="curType != 'view' && addOrEditForm.specifyVersionType == '0'" class='operBtn' @click='clearVersionBtnClick' style='margin-right: 10px;'><i class='el-icon el-icon-operation-clear commonTextNormal12'></i></div>
												<div v-show="curType != 'view' && addOrEditForm.specifyVersionType == '0'" class='operBtn' @click='addVersionClick' style='margin-right: 10px;'><i class='el-icon el-icon-plus commonTextNormal12'></i></div>
											</div>
										</div>
 										<el-table-column label="<%=rb.getString("ChuShiBanBen") %>" prop="originalVersion" show-overflow-tooltip="true">
											<template slot-scope="scope">
												<div class='table-suffix'>
						                        	<span class="text">{{scope.row.originalVersion}} </span>
						                        	<span v-show="curType != 'view' && addOrEditForm.specifyVersionType=='0'"><i class='el-icon el-icon-circle-close ipTextWarp deleteVersion' @click='deleteVersionItem(scope.row)'></i></span>
						                        </div>
											</template>
										</el-table-column>
									</el-ctable>
								</div><!-- 模板版本和保留配置 -->
								<div style='padding: 60px 60px 0;'>
									<el-form-item label="<%=rb.getString("MuBiaoBanBen") %>" label-position="top" prop="targetVersion" class="selectVersionBox" style=" margin-bottom: 16px;">
					                    <el-select v-model="addOrEditForm.targetVersion" :disabled="isReadOnly" placeholder="<%=rb.getString("QingXuanZe") %>"> 
					                       <el-option v-for="item in tarVersionList" :key="item.text" :label="item.text" :value="item.text"></el-option>
					                    </el-select>
					                </el-form-item> 
					                <el-checkbox v-model="addOrEditForm.preserveSetting" true-label="1" false-label="0" :disabled="isReadOnly"></el-checkbox>
					                <span class='commonSize14' style='margin-left: 6px;'><%=rb.getString("eNBBaoLiuPeiZhi") %></span>
								</div>
							</div> 
							<el-form-item label-width='0' prop="originalVersion">
			                	<el-input v-model="addOrEditForm.originalVersion" v-show=false></el-input>
			                </el-form-item> 							
						</div>        
			        </div><!-- License  -->
			        <div v-show="functionModulesSelect == '1'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend">
							<span class="title-icon"></span>
							<span class="title-text">License</span>
							<el-switch v-model="addOrEditForm.licenseEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="isReadOnly"></el-switch>
						</div>
			            <div style='height: 334px; padding-left: 26px;'>
							<el-ctable :height="height" ref="licenseTable" id='licenseTable' :url="licenseUrl" :time="6" :row-key="'serial_number'" class='commonBorderRadius commonBorder' style='margin-top: 10px; margin-bottom: 10px;'
								:pagination="true" 
								:query-params="enbLicenseQuery"
								row-key="serial_number">
								<div slot="toolbar">
									<div class='commonFlex commonToolBarBox commonContent licenseBox'>
										<span class='commonTitle12' style='padding: 6px 20px;'><%=rb.getString("DaoRuLicenseFile")%></span>
										<div class='commonFlex'>
											<el-query type="normal" @query="queryEnbLicense" placeholder="<%=rb.getString("XiaoZhanBianMa")%>" style='margin-right: 10px;'></el-query>
											<div v-show='curType != "view"' class='licenseImportBox' @click='importLicenseClick'><i class='el-icon el-icon-operation-import importIcon'></i></div>
										</div>
									</div>
								</div>
								<el-table-column width="40" v-if='curType != "view"'>
									<template slot-scope="scope">
										<div><i @click="deleteLicenseClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon"></i></div>
									</template>
								</el-table-column>
								<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number"></el-table-column>
								<el-table-column label="<%=rb.getString("LicenseWenJian")%>" prop="file_name"></el-table-column>
								<el-table-column label="<%=rb.getString("ShangChuanShiJian")%>" prop="upload_time"></el-table-column>
								<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="execute_status" :formatter="executeStatus"></el-table-column>
							</el-ctable>
			            </div>
			        </div><!-- Parametes Configuration -->
                 	<div v-show="functionModulesSelect == '2'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend">
							<span class="title-icon"></span>
							<span class="title-text"><%=rb.getString("ENBCanShuZiPeiZhi") %></span>
							<el-switch v-model="addOrEditForm.selfConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="isReadOnly"></el-switch>
						</div>
			            <div style='padding-left: 26px; padding-top: 16px; '>
                            <span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 20px; margin-top: -6px;'><%=rb.getString("ENBCanShuPeiZhiTiShi") %></span>
                            <div class='reginDeployingBox infoSpecifiedDevice' style=' width: 100%; padding-bottom: 100px;'>
                                <el-tabs v-model="parameterConfigActive" @tab-click="clickTab" type='border-card' style='height: auto' >
                                    <el-tab-pane label="<%=rb.getString("CanShuShuJuChi") %>" name="paramOne" style="position:relative">
                                        <div class='paramTwoBox'>                                       	
                                            <el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='switchEnable' label-width='80px' class="enableCommon basicLeftLabel" style='margin-bottom: 0;'>
                								<el-switch v-model="addOrEditForm.switchEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="isReadOnly"></el-switch>
                                            </el-form-item>
					   						<el-collapse v-model="activeNameOne">
												<el-collapse-item name="basic">
													<template slot='title'>
														<p style="display:inline-block;margin-left: 26px;"><span class='commonTextWeight14'><%=rb.getString("ENBJiChuPeiZhi")%></span></p>
													</template>
													<div class='paramsItemWarp' style='margin-left: 26px; margin-right: 0px;'>
														<div v-show="curProductName == 'QAFA'">
															<div class='commonFlex commonContent'>
																<div class='commonFlex'>
																	<span class='commonTitle12' style='padding: 6px 0;'><%=rb.getString("MoKuaiLeiXingLieBiao") %></span>
																	<el-checkbox v-model="addOrEditForm.module_enable" label=" " true-label="1" false-label="0" :disabled="isReadOnly" style='margin: 10px 0 0 10px;'></el-checkbox>																	
																</div>
																<span @click='addModuleBtnClick' v-show='curType != "view"' class='licenseImportBox'><i class='el-icon el-icon-plus importIcon' style='margin: 0;'></i></span>																												
															</div>
															<el-ctable ref="ctableModule" :data="addOrEditForm.moduleTypeList" height='200px' :pagination="false" class='commonBorderRadius commonBorder' style='margin-bottom: 26px;'>
																<el-table-column width="70" v-if='curType != "view"'>
																	<template slot-scope="scope">
																		<div>
																			<i @click="moduleModifyClick(scope.row, scope.$index)" class="el-icon el-icon-operation-edit commonIcon"></i>
																			<i @click="moduleDeleteClick(scope.row, scope.$index)" class="el-icon el-icon-operation-delete commonIcon" style='margin-left: 6px;'></i>
																		</div>
																	</template>
																</el-table-column>
																<el-table-column label="<%=rb.getString("SheBeiXingHaoMing") %>" prop="module_type"></el-table-column>
																<el-table-column label="<%=rb.getString("DaiKuan") %>" prop="band_width" :formatter="bandwidthFmtOne"></el-table-column>
																<el-table-column label="<%=rb.getString("ENBPinDian")%>" prop="frequency"></el-table-column>
															</el-ctable>
															<el-form-item v-show="false" prop="moduleTypeList">
																<el-input v-model="addOrEditForm.moduleTypeList"></el-input>
															</el-form-item>	
															<el-form-item v-show="false" prop="module_enable">
																<el-input v-model="addOrEditForm.module_enable"></el-input>
															</el-form-item>															
														</div>

														<el-form-item label='<%=rb.getString("ZhiChiPinDuan")%>' prop="bands_support" class='inputCommon validateItem'>
															<el-input v-model.trim="addOrEditForm.bands_support" :disabled="isReadOnly">
																<template slot="append" v-if='curProductName == "DXDF"'><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 1~85</template>
																<template slot="append" v-if='curProductName != "DXDF"'><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 1~62</template>
															</el-input>
														</el-form-item>
														<el-form-item label='<%=rb.getString("DaiKuan")%>' prop="band_width" v-show='curProductName == "DXDF"'>
															<el-select v-model="addOrEditForm.band_width" :disabled="isReadOnly">
																<el-option label="6" value="6"></el-option>
																<el-option label="15" value="15"></el-option>
																<el-option label="25" value="25"></el-option>
																<el-option label="50" value="50"></el-option>
																<el-option label="75" value="75"></el-option>
																<el-option label="100" value="100"></el-option>
															</el-select>	
														</el-form-item>
														<el-form-item label='<%=rb.getString("DaiKuan")%>' prop="band_width" v-show='curProductName != "NBIOT" && curProductName != "DXDF"'>
															<el-select v-model="addOrEditForm.band_width" :disabled="isReadOnly">
																<el-option label="5MHz" value="n25"></el-option>
																<el-option label="10MHz" value="n50"></el-option>
																<el-option label="15MHz" value="n75"></el-option>
																<el-option label="20MHz" value="n100"></el-option>
															</el-select>
														</el-form-item>
														<el-form-item label='<%=rb.getString("ENBPinDian")%>' prop="frequency" v-show='curProductName == "DXDF"' class='validateItem'>
															<el-input v-model="addOrEditForm.frequency" :disabled="isReadOnly"></el-input>
															<span class='commonTitle12 commonLeft10'><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 0-262143</span>
														</el-form-item>
														<el-form-item label='<%=rb.getString("ENBPinDian")%>' prop="frequency" v-show='curProductName != "NBIOT" && curProductName != "DXDF"' class='validateItem'>
															<el-input v-model.trim="addOrEditForm.frequency" @blur="changeToFrequencyOne('dl')" @focus="changeToEarfcnOne('dl')" :disabled="isReadOnly">
																<template slot="append"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 1~65535</template>
															</el-input>
														</el-form-item>
														<el-form-item label="DL Frequency" prop="frequency" v-show='curProductName == "NBIOT" && curProductName != "DXDF"' class='validateItem'>
															<el-input v-model.trim="addOrEditForm.frequency" @blur="changeToFrequencyOne('dl')" @focus="changeToEarfcnOne('dl')" :disabled="isReadOnly">
																<template slot="append"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 1~65535</template>
															</el-input>
														</el-form-item>
														<el-form-item label='<%=rb.getString("ZiZhenPeiBi")%>' prop="subframe_assignment" v-show='curProductName != "NBIOT" && curProductName != "QAFA" && curProductName != "QAFB" && curProductName != "BAIBLQ" && curProductName != "DXDF"'>
															<el-select v-model="addOrEditForm.subframe_assignment" :disabled="isReadOnly">
																<el-option label='<%=rb.getString("QingXuanZe")%>' value=""></el-option>
																<el-option label="0(DL:UL = 1:3)" value="0" v-show='curProductName == "RTS" || curProductName == "RTD"'></el-option>
																<el-option label="1(DL:UL = 2:2)" value="1"></el-option>
																<el-option label="2(DL:UL = 3:1)" value="2"></el-option>
																<el-option label='6(DL:UL = 3:5)' value='6' v-show='curProductName == "QRTB" || curProductName == "BAIBLQ" || curProductName == "MLQ" || curProductName == "CR-B4860" || curProductName == "MLN"'></el-option>
															</el-select>
														</el-form-item>
														<el-form-item label='<%=rb.getString("TeShuZiZhenPeiBi")%>' prop="special_subframe_patterns" v-show='curProductName != "NBIOT" && curProductName != "QAFA" && curProductName != "QAFB" && curProductName != "BAIBLQ"  && curProductName != "DXDF"'>
															<el-select v-model="addOrEditForm.special_subframe_patterns" :disabled="isReadOnly">
																<el-option label='<%=rb.getString("QingXuanZe")%>' value=""></el-option>
																<el-option label="5" value="5"></el-option>
																<el-option label="7" value="7"></el-option>
															</el-select>
														</el-form-item>
														<el-form-item label='<%=rb.getString("TAC")%>' prop="tac" class='validateItem'>
															<el-input v-model.trim="addOrEditForm.tac" :disabled="isReadOnly">
																<template slot="append"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 0~65535</template>
															</el-input>
														</el-form-item>
														<el-form-item :label="eciLable" prop="cell_identity" class='validateItem'>
															<el-input v-model.trim="addOrEditForm.cell_identity" :disabled="isReadOnly">
																<template slot="append"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 0~268435455</template>
															</el-input>
														</el-form-item>
														<el-form-item label="PCI" prop="phycellid" class='validateItem'>
															<el-input v-model.trim="addOrEditForm.phycellid" :disabled="isReadOnly" class='validateItem'>
																<template slot="append"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 0~503</template>
															</el-input>
														</el-form-item>
														<div v-show='curProductName == "DXDF"' class='paramPoolWarp'>
															<el-form-item label='PLMN ID' prop="plmn_id">
																<el-input v-model="addOrEditForm.plmn_id" :disabled="isReadOnly"></el-input>
																<span class='commonTitle12 commonLeft10'><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 00000-999999</span>
															</el-form-item>
															<el-form-item label='<%=rb.getString("CPETxPower")%>' prop="txPower">
																<el-select v-model="addOrEditForm.txPower" filterable multiple collapse-tags>
																	<el-option v-for="item in dxdfPowerList" :key="item.value" :label="item.label" :value="item.value"></el-option>
																</el-select>
															</el-form-item>
														</div>
														<div v-show='curProductName != "DXDF"' style='width: 100%;' class='paramPoolWarp'>
															<el-form-item prop="root_sequence_index" label='<%=rb.getString("GenXuLieSuoYin")%>' v-show='curProductName != "NBIOT"' class='validateItem'>
																<el-input v-model.trim="addOrEditForm.root_sequence_index" :disabled="isReadOnly">
																	<template slot="append"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 0~837</template>
																</el-input>
															</el-form-item>
															<el-form-item prop="carrier_mode" label='<%=rb.getString("ZaiBoLeiXing")%>' v-show='curProductName == "RTD"'>
																<el-select v-model="addOrEditForm.carrier_mode" :disabled="isReadOnly">
																	<el-option label='<%=rb.getString("QingXuanZe")%>' value=""></el-option>
																	<el-option label='<%=rb.getString("DanZaiBo")%>' value="1"></el-option>
																	<el-option label='<%=rb.getString("ShuangZaiBo")%>' value="2"></el-option>
																</el-select>
															</el-form-item>	
														</div>
													</div>				
												</el-collapse-item>
												<!-- 所有产品类型都有的模块： halob_enable(mme), plmn-->
												<el-collapse-item name="coreNetwork" v-if='curProductName != "DXDF"'>
													<template slot='title'>
														<p style="display:inline-block;margin-left: 26px;"><span class='commonTextWeight14'><%=rb.getString("CoreNetwork")%></span></p>
													</template>
													<div style='margin-left: 26px; margin-right: 0px;' class='paramPoolWarp'>
														<el-form-item label="<%=rb.getString("HaloBKaiGuan")%>" prop="halob_enable">
															<el-select v-model="addOrEditForm.halob_enable"  :disabled="isReadOnly" placeholder="<%=rb.getString("QingXuanZe") %>">
																<el-option label="<%=rb.getString("HalobKaiQi")%>" value="1"></el-option>
																<el-option label="<%=rb.getString("HalobGuanBi")%>" value="0"></el-option>
															</el-select>
														</el-form-item>
														<el-form-item label="<%=rb.getString("PLMN")%>" prop="plmn_id" class='validateItem'>
															<el-input v-model.trim="addOrEditForm.plmn_id" :disabled="isReadOnly">
																<template slot="append"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 00000~999999</template>
															</el-input>
														</el-form-item>
														<el-form-item v-show="showMMEMain" label="MME" style="position: relative; margin-bottom: 0px;">
															<el-input v-model="addOrEditForm.mme" :disabled="isReadOnly"></el-input>
															<span v-show="curType != 'view'" @click="addMMEOne" class="el-icon el-icon-plus" style="position: absolute;left: 210px;top: 5px;"></span>
															<div class='versionResultBox' v-show='addOrEditForm.mmeGroup.length > 0' style='max-height: 96px; width: 198px;'>
																<div class='suffixItem' v-for='(domain,index) in addOrEditForm.mmeGroup'>
																	<div class='form-suffix' style='width: 180px; margin: 3px 0 0 10px;'>
																		<span class='text' style='width: 160px;'>{{domain}}</span>
																		<span v-show="curType != 'view'"><i class='el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='removeMMEOne(domain)'></i></span>
																	</div>
																</div>
															</div>
															<p class='ipErrorTip'>{{errorMsgOne}}</p>
														</el-form-item>
														<el-form-item prop="mmeStr" style="margin-right:0px;" v-show=false>
															<el-input v-model="addOrEditForm.mmeStr"></el-input>
														</el-form-item>
													</div>
												</el-collapse-item>
											</el-collapse>
	                                        <!-- advance -->
	                                        <div v-if="groupShow.wan || groupShow.ipsec || groupShow.power || groupShow.dns || groupShow.ntp">
							   					<el-form ref="advance" :model="advanceForm" label-position="top">
												<el-collapse>
													<el-collapse-item name="advance">
														<template slot='title'>
															<p style="display:inline-block;margin-left: 26px;">
																<span class='commonTextWeight14'><%=rb.getString("ENBGaoJiSheZhi")%></span>
																<el-switch 
																	onclick="event.stopPropagation()"
																	v-model="advanceForm.advanceEnable" 
																	active-value="1" inactive-value="0" 
																	:disabled="isReadOnly" style="margin-left: 20px;">
																</el-switch>
															</p>
														</template>
														<div style='margin-left: 26px; margin-right: 60px;'>
															<div class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='commonGeneralBold12 ' style='padding: 6px 0;'>Parameters Configure</span>	
																<span class='commonTitle12' style='padding: 6px;'>You can select the parameters in the parameter list on the right to configure.</span>
																<span v-if="!isReadOnly" @click='advanceSettingClick' class='licenseImportBox'><i class='el-icon el-icon-plus importIcon'></i></span>																												
															</div>
															<div v-if="advanceTreeSelected.includes('wanConfig') && groupShow.wan" class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='commonTextWeight14' style='padding: 6px 0;'>WAN Config</span>		
																<span v-if="!isReadOnly" @click="removeGroupItem('wanConfig')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>																											
															</div>
															<div v-if="advanceTreeSelected.includes('wanConfig') && groupShow.wan" style='width: 100%; ' class='paramPoolWarp'>
																<el-form-item label="MTU" v-if="paramKeys.includes('mtu')"
																	prop="mtu"
																	:rules="{
																		validator: function(rule, val, cb){
																			var reg = /^(\d+\.\.){0,1}(\d+)$/;
																			if(!reg.test(val) || val-700<0 || val-1600>0) {
																				cb('Range: 700 - 1600');
																			}else {
																				cb();
																			}
																		}
																	}">
																	<el-input v-model="advanceForm.mtu" :disabled="isReadOnly"></el-input>
																</el-form-item>
															</div>
															<div v-if="advanceTreeSelected.includes('ipsecSettings') && groupShow.ipsec" class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='commonTextWeight14' style='padding: 6px 0;'>IPSec Settings</span>		
																<span v-if="!isReadOnly" @click="removeGroupItem('ipsecSettings')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>
																<!-- 高通站： QAFA QATA QAFB， 后端支持下发一组， 基站 LMT 可配置3组；Intel 站： RTS QRTB 4860 RTD 支持下发2组；最终处理结果： 前后端需保持一致；ipsec: ['RTS','RTD','QRTB','QAFA','QATA','QAFB','PM-B4860','CR-B4860/BU','CR-B4860/EU','CR-B4860/RU'].includes(type) -->
																<span v-if="!isReadOnly && advanceForm.ipsecList.length<1 && (curProductName == 'QAFA' || curProductName == 'QATA'  || curProductName == 'QAFB')" @click="addIpsecItem" class='licenseImportBox'><i class='el-icon el-icon-plus importIcon'></i></span>
																<span v-if="!isReadOnly && advanceForm.ipsecList.length<2 && (curProductName == 'RTS' || curProductName == 'QRTB' || curProductName == 'BAIBLQ' || curProductName == 'BLX'|| curProductName == 'MLQ' || curProductName == 'CR-B4860' || curProductName == 'RTD' || curProductName == 'MLN' )" @click="addIpsecItem" class='licenseImportBox'><i class='el-icon el-icon-plus importIcon'></i></span>
															</div>
															<div v-if="advanceTreeSelected.includes('ipsecSettings') && groupShow.ipsec" style='width: 100%; margin-bottom: 20px;' class='paramPoolWarp'>
																<el-form-item label="IPSEC Enable" style="display: flex;margin: 10px 0 0 0;">
																	<el-switch v-model="advanceForm.ipsec_enable" active-value="1" inactive-value="0" :disabled="isReadOnly" style="margin-left: 20px;"></el-switch>
																</el-form-item>
																<fieldset :flag="addKey(item.IPSEC_INDEX)" :class="{'fielset-cls': true, 'extended': extend['index_'+item.IPSEC_INDEX]}" v-for="(item,index) in advanceForm.ipsecList">
																	<legend>
																		<div style="position: relative; border: 1px solid #e9e9e9;border-radius: 5px;margin: 0px 15px;padding: 0px 25px 0px 10px;">
																			<span @click="extend['index_'+item.IPSEC_INDEX] = !extend['index_'+item.IPSEC_INDEX]">Tunnel {{index+1}}</span>
																			<i v-if="!isReadOnly" @click="removeIpsec(index)" class="el-icon el-icon-close" style="zoom: 0.3;top: 25px;right: 30px;"></i>
																		</div>
																	</legend>
																	<el-form-item label="Enable" v-if="paramKeys.includes('TUNNEL_ENABLE')">
																		<el-switch v-model="item.TUNNEL_ENABLE" active-value="1" inactive-value="0" :disabled="isReadOnly"></el-switch>
																	</el-form-item>
																	<el-form-item label="AuthBy" v-if="paramKeys.includes('authBy')">
																		<el-select v-model="item.authBy" :disabled="isReadOnly">
																			<el-option label="psk" value="psk"></el-option>
																			<el-option label="cert" value="cert"></el-option>
																			<el-option label="aka_psk" value="aka_psk"></el-option>
																			<el-option label="aka_cert" value="aka_cert"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="leftAuth" v-if="paramKeys.includes('LEFT_AUTH')">
																		<el-select v-model="item.LEFT_AUTH" :disabled="isReadOnly">
																			<el-option label="psk" value="psk"></el-option>
																			<el-option label="pubkey" value="pubkey"></el-option>
																			<el-option label="eap-aka" value="eap-aka"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="rightAuth" v-if="paramKeys.includes('RIGHT_AUTH')">
																		<el-select v-model="item.RIGHT_AUTH" :disabled="isReadOnly">
																			<el-option label="psk" value="psk"></el-option>
																			<el-option label="pubkey" value="pubkey"></el-option>
																			<el-option label="eap-aka" value="eap-aka"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="Gateway" v-if="paramKeys.includes('TUNNEL_GATEWAY')"
																		:prop="'ipsecList.'+index+'.TUNNEL_GATEWAY'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(noZh(val)) {
																						cb();
																					}else {
																						cb(msg.noZh+'0-64');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.TUNNEL_GATEWAY" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="leftId" v-if="paramKeys.includes('LEFT_IDENTIFIER')"
																		:prop="'ipsecList.'+index+'.LEFT_IDENTIFIER'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(noZh(val)) {
																						cb();
																					}else {
																						cb(msg.noZh+'0-64');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.LEFT_IDENTIFIER" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="rightId" v-if="paramKeys.includes('RIGHT_IDENTIFIER')"
																		:prop="'ipsecList.'+index+'.RIGHT_IDENTIFIER'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(noZh(val)) {
																						cb();
																					}else {
																						cb(msg.noZh+'0-64');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.RIGHT_IDENTIFIER" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="leftCert" v-if="paramKeys.includes('LEFT_CERT')"
																		:prop="'ipsecList.'+index+'.LEFT_CERT'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(noZh(val)) {
																						cb();
																					}else {
																						cb(msg.noZh+'0-64');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.LEFT_CERT" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="secretKey" v-if="paramKeys.includes('SECRET_KEY')"
																		:prop="'ipsecList.'+index+'.SECRET_KEY'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(noZh(val)) {
																						cb();
																					}else {
																						cb(msg.noZh+'0-64');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.SECRET_KEY" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="rightSecretKey" v-if="paramKeys.includes('RIGHT_SECRET_KEY')"
																		:prop="'ipsecList.'+index+'.RIGHT_SECRET_KEY'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(noZh(val)) {
																						cb();
																					}else {
																						cb(msg.noZh+'0-64');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.RIGHT_SECRET_KEY" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="leftSourceIp" v-if="paramKeys.includes('LEFTSOURCEIP')"
																		:prop="'ipsecList.'+index+'.LEFTSOURCEIP'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(isValidIP(val) || '%config' == val) {
																						cb();
																					}else {
																						cb('<%=rb.getString("IPDiZhiHuoConfig")%>');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.LEFTSOURCEIP" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="leftSubnet" v-if="paramKeys.includes('LEFT_SUBNET')"
																		:prop="'ipsecList.'+index+'.LEFT_SUBNET'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(noZh(val)) {
																						cb();
																					}else {
																						cb(msg.noZh+'0-64');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.LEFT_SUBNET" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="rightSubnet" v-if="paramKeys.includes('RIGHT_SUBNET')"
																		:prop="'ipsecList.'+index+'.RIGHT_SUBNET'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(noZh(val)) {
																						cb();
																					}else {
																						cb(msg.noZh+'0-64');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.RIGHT_SUBNET" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="IKE Encryption" v-if="paramKeys.includes('IKE_ENCRYPTION')">
																		<el-select v-model="item.IKE_ENCRYPTION" :disabled="isReadOnly">
																			<el-option label="aes128" value="aes128"></el-option>
																			<el-option label="aes256" value="aes256"></el-option>
																			<el-option label="3des" value="3des"></el-option>
																			<el-option label="des" value="des"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="IKE DH Group" v-if="paramKeys.includes('IKE_DH_GROUP')">
																		<el-select v-model="item.IKE_DH_GROUP" :disabled="isReadOnly">
																			<el-option label="modp768" value="modp768"></el-option>
																			<el-option label="modp1024" value="modp1024"></el-option>
																			<el-option label="modp1536" value="modp1536"></el-option>
																			<el-option label="modp2048" value="modp2048"></el-option>
																			<el-option label="modp4096" value="modp4096"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="IKE Authentication" v-if="paramKeys.includes('IKE_AUTHENTICATION')">
																		<el-select v-model="item.IKE_AUTHENTICATION" :disabled="isReadOnly">
																			<el-option label="sha1" value="sha1"></el-option>
																			<el-option label="sha1_160" value="sha1_160"></el-option>
																			<el-option label="sha256_96" value="sha256_96"></el-option>
																			<el-option label="sha256" value="sha256"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="ESP Encryption" v-if="paramKeys.includes('ESP_ENCRYPTION')">
																		<el-select v-model="item.ESP_ENCRYPTION" :disabled="isReadOnly">
																			<el-option label="aes128" value="aes128"></el-option>
																			<el-option label="aes256" value="aes256"></el-option>
																			<el-option label="3des" value="3des"></el-option>
																			<el-option label="des" value="des"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="ESP DH Group" v-if="paramKeys.includes('ESP_DH_GROUP')">
																		<el-select v-model="item.ESP_DH_GROUP" :disabled="isReadOnly">
																			<el-option label="modp768" value="modp768"></el-option>
																			<el-option label="modp1024" value="modp1024"></el-option>
																			<el-option label="modp1536" value="modp1536"></el-option>
																			<el-option label="modp2048" value="modp2048"></el-option>
																			<el-option label="modp4096" value="modp4096"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="ESP Authentication" v-if="paramKeys.includes('ESP_AUTHENTICATION')">
																		<el-select v-model="item.ESP_AUTHENTICATION" :disabled="isReadOnly">
																			<el-option label="sha1" value="sha1"></el-option>
																			<el-option label="sha1_160" value="sha1_160"></el-option>
																			<el-option label="sha256_96" value="sha256_96"></el-option>
																			<el-option label="sha256" value="sha256"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="IKELifeTime" v-if="paramKeys.includes('IKELIFETIME')"
																		:prop="'ipsecList.'+index+'.IKELIFETIME'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(smhd(val)) {
																						cb();
																					}else {
																						cb(msg.smhd);
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.IKELIFETIME" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="KeyLife" v-if="paramKeys.includes('KEYLIFE')"
																		:prop="'ipsecList.'+index+'.KEYLIFE'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(smhd(val)) {
																						cb();
																					}else {
																						cb(msg.smhd);
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.KEYLIFE" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="RekeyMargin" v-if="paramKeys.includes('REKEYMARGIN')"
																		:prop="'ipsecList.'+index+'.REKEYMARGIN'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(smhd(val)) {
																						cb();
																					}else {
																						cb(msg.smhd);
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.REKEYMARGIN" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="Dpdaction" v-if="paramKeys.includes('DPDACTION')">
																		<el-select v-model="item.DPDACTION" :disabled="isReadOnly">
																			<el-option label="none" value="none"></el-option>
																			<el-option label="clear" value="clear"></el-option>
																			<el-option label="hold" value="hold"></el-option>
																			<el-option label="restart" value="restart"></el-option>
																		</el-select>
																	</el-form-item>
																	<el-form-item label="Dpddelay" v-if="paramKeys.includes('DPDDELAY')"
																		:prop="'ipsecList.'+index+'.DPDDELAY'"
																		:rules="{
																			validator: function(rule, val, cb){
																				if(val) {
																					if(smhd(val)) {
																						cb();
																					}else {
																						cb('<%=rb.getString("ShuZiJiaSMHD")%>');
																					}
																				}else {
																					cb();
																				}
																			}
																		}">
																		<el-input v-model="item.DPDDELAY" maxlength="64" :disabled="isReadOnly"></el-input>
																	</el-form-item>
																</fieldset>
															</div>
															<!-- POWER -->
															<div v-if="advanceTreeSelected.includes('powerControl') && groupShow.power" class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='commonTextWeight14' style='padding: 6px 0;'>Power Control Parameters</span>	
																<span v-if="!isReadOnly" @click="removeGroupItem('powerControl')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>																													
															</div>
															<div v-if="advanceTreeSelected.includes('powerControl') && groupShow.power" style='width: 100%; ' class='paramPoolWarp'>
																<el-form-item label="Total Tx Power" v-if="paramKeys.includes('totalTxPower') && (curProductName =='QAFA' || curProductName =='QATA')">
																	<el-select v-model="advanceForm.totalTxPower" :disabled="isReadOnly" filterable>
																		<el-option v-for="item in powerList" :key="item.value" :label="item.label" :value="item.value"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label="Power Ramping" v-if="paramKeys.includes('powerRamping')">
																	<el-select v-model="advanceForm.powerRamping" :disabled="isReadOnly">
																		<el-option label="0" value="0"></el-option>
																		<el-option label="2" value="2"></el-option>
																		<el-option label="4" value="4"></el-option>
																		<el-option label="6" value="6"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label="Preamble Init Target Power" v-if="paramKeys.includes('preambleInitTargetPower')">
																	<el-select v-model="advanceForm.preambleInitTargetPower" :disabled="isReadOnly">
																		<el-option label="-120" value="-120"></el-option>
																		<el-option label="-118" value="-118"></el-option>
																		<el-option label="-116" value="-116"></el-option>
																		<el-option label="-114" value="-114"></el-option>
																		<el-option label="-112" value="-112"></el-option>
																		<el-option label="-110" value="-110"></el-option>
																		<el-option label="-108" value="-108"></el-option>
																		<el-option label="-106" value="-106"></el-option>
																		<el-option label="-104" value="-104"></el-option>
																		<el-option label="-102" value="-102"></el-option>
																		<el-option label="-100" value="-100"></el-option>
																		<el-option label="-98" value="-98"></el-option>
																		<el-option label="-96" value="-96"></el-option>
																		<el-option label="-94" value="-94"></el-option>
																		<el-option label="-92" value="-92"></el-option>
																		<el-option label="-90" value="-90"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label="Po_nominal_pusch" v-if="paramKeys.includes('poNominalPusch')"
																	prop="poNominalPusch"
																	:rules="{
																		validator: function(rule, value, cb){
																			var max = 24,
																				min = -126,
																				msg = 'Range: ' + (min-0<0?'('+min+')':min) + ' - ' + (max-0<0?'('+max+')':max),
																				reg = /^-?\d+$/;
																				if(value){
																					if(reg.test(value) && parseInt(value)>= -126 && parseInt(value)<= 24) {
																						cb();
																					}else {
																						cb(msg);
																					}
																				}else{
																					cb();
																				}
																		}
																	}">
																	<el-input v-model="advanceForm.poNominalPusch" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="Po_nominal_pucch" v-if="paramKeys.includes('poNominalPucch')"
																	prop="poNominalPucch"
																	:rules="{
																		validator: function(rule, value, cb){
																			var max = -96,
																				min = -127,
																				msg = 'Range: ' + (min-0<0?'('+min+')':min) + ' - ' + (max-0<0?'('+max+')':max),
																				reg = /^-?\d+$/;

																			if(value){
																				if(reg.test(value) && parseInt(value)>= -127 && parseInt(value)<= -96) {
																					cb();
																				}else {
																					cb(msg);
																				}
																			}else{
																				cb();
																			}
																		}
																	}">
																	<el-input v-model="advanceForm.poNominalPucch" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="alpha" v-if="paramKeys.includes('alpha')">
																	<el-select v-model="advanceForm.alpha" :disabled="isReadOnly">
																		<el-option label="0" value="0"></el-option>
																		<el-option label="40" value="40"></el-option>
																		<el-option label="50" value="50"></el-option>
																		<el-option label="60" value="60"></el-option>
																		<el-option label="70" value="70"></el-option>
																		<el-option label="80" value="80"></el-option>
																		<el-option label="90" value="90"></el-option>
																		<el-option label="100" value="100"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label="Target ul sinr" v-if="paramKeys.includes('targetUlSinr')"
																	prop="targetUlSinr"
																	:rules="{
																		validator: function(rule, value, cb){
																			var selected = productList.filter(function(item){
																					return item.value == addOrEditForm.productType;
																				})[0],
																				type = selected?selected.name:'',
																				reg = /^-?\d+$/;

																			var max = 10,
																				min = -6;
																			
																			if(['QAFA','QATA','QAFB'].includes(type)) {
																				max = 255,
																				min = 0;
																			}
																			var msg = 'Range: ' + (min-0<0?'('+min+')':min) + ' - ' + (max-0<0?'('+max+')':max);
																			if(value){
																				if(reg.test(value) && parseInt(value)>=min && parseInt(value)<=max) {
																					cb();
																				}else {
																					cb(msg);
																				}
																			}else{
																				cb();
																			}
																		}
																	}">
																	<el-input v-model="advanceForm.targetUlSinr" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="PA" v-if="paramKeys.includes('powerPA')">
																	<el-select v-model="advanceForm.powerPA" :disabled="isReadOnly">
																		<el-option label="-6dB" value="-600"></el-option>
																		<el-option label="-4.77dB" value="-477"></el-option>
																		<el-option label="-3dB" value="-300"></el-option>
																		<el-option label="-1.77dB" value="-177"></el-option>
																		<el-option label="0dB" value="0"></el-option>
																		<el-option label="1dB" value="100"></el-option>
																		<el-option label="2dB" value="200"></el-option>
																		<el-option label="3dB" value="300"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label="PB" v-if="paramKeys.includes('powerPB')" 
																	prop="powerPB"
																	:rules="{
																		validator: function(rule, val, cb){
																			var reg = /^(\d+\.\.){0,1}(\d+)$/;
																			if(val){
																				if(reg.test(val) && parseInt(val)>=0 && parseInt(val)<=3) {
																					cb();
																				}else {
																					cb('Range: 0 - 3');
																				}
																			}else{
																				cb();
																			}
																		}
																	}">
																	<el-input v-model="advanceForm.powerPB" :disabled="isReadOnly"></el-input>
																</el-form-item>
															</div>
															<!-- DNS -->
															<div v-if="advanceTreeSelected.includes('dns') && groupShow.dns" class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='commonTextWeight14' style='padding: 6px 0;'>DNS</span>		
																<span v-if="!isReadOnly" @click="removeGroupItem('dns')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>																												
															</div>
															<div v-if="advanceTreeSelected.includes('dns') && groupShow.dns" style='width: 100%; ' class='paramPoolWarp'>
																<el-form-item label="Host Name" v-if="paramKeys.includes('dnsHostName')">
																	<el-input v-model="advanceForm.dnsHostName" maxlength="100" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="TimeZone" v-if="paramKeys.includes('dnsTimeZone') && (curProductName =='QAFA' || curProductName =='QATA')">
																	<el-select v-model="advanceForm.dnsTimeZone" filterable :disabled="isReadOnly">
																		<el-option v-for="item in timeZoneList" :key="item.value" :label="item.label" :value="item.value"></el-option>
																	</el-select>
																</el-form-item>
																<el-form-item label="DNS Address 1" v-if="paramKeys.includes('dnsAddress1')"
																	prop="dnsAddress1"
																	:rules="{
																		validator: function(rule, val, cb){
																			if(val) {
																				if(isValidIP(val)) {
																					cb();
																				}else {
																					cb('IP Address');
																				}
																			}else {
																				cb();
																			}
																		}
																	}">
																	<el-input v-model="advanceForm.dnsAddress1" maxlength="50" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="DNS Address 2" v-if="paramKeys.includes('dnsAddress2')"
																	prop="dnsAddress2"
																	:rules="{
																		validator: function(rule, val, cb){
																			if(val) {
																				if(isValidIP(val)) {
																					cb();
																				}else {
																					cb('IP Address');
																				}
																			}else {
																				cb();
																			}
																		}
																	}">
																	<el-input v-model="advanceForm.dnsAddress2" maxlength="50" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="DNS Address 3" v-if="paramKeys.includes('dnsAddress3')"
																	prop="dnsAddress3"
																	:rules="{
																		validator: function(rule, val, cb){
																			if(val) {
																				if(isValidIP(val)) {
																					cb();
																				}else {
																					cb('IP Address');
																				}
																			}else {
																				cb();
																			}
																		}
																	}">
																	<el-input v-model="advanceForm.dnsAddress3" maxlength="50" :disabled="isReadOnly"></el-input>
																</el-form-item>
															</div>
															<!-- NTP 
																	QRTB 基站只有一个 port, 4860 站 没有port, port2 port3 删除； 后端下发也只是一个port
															-->
															<div v-if="advanceTreeSelected.includes('ntp') && groupShow.ntp" class='commonFlex commonContent' style="justify-content: flex-start;">
																<span class='commonTextWeight14' style='padding: 6px 0;'>NTP</span>			
																<span v-if="!isReadOnly" @click="removeGroupItem('ntp')" class='licenseImportBox'><i class='el-icon el-icon-operation-delete importIcon'></i></span>																											
															</div>
															<div v-if="advanceTreeSelected.includes('ntp') && groupShow.ntp" style='width: 100%; ' class='paramPoolWarp'>
																<el-form-item label="Enable" v-if="paramKeys.includes('ntpEnable')">
																	<el-switch v-model="advanceForm.ntpEnable" active-value="1" inactive-value="0" :disabled="isReadOnly"></el-switch>
																</el-form-item>
																<el-form-item label="Port1" v-if="paramKeys.includes('ntpPort1')" prop="ntpPort1" :rules="{
																	validator: function(rule, val, cb){
																		var reg = /^(\d+\.\.){0,1}(\d+)$/;
																		if(val){
																			if(reg.test(val) && parseInt(val)>=1 && parseInt(val)<=65535) {
																				cb();
																			}else {
																				cb('Range: 1 - 65535');
																			}
																		}else{
																			cb();
																		}
																	}
																}">
																	<el-input v-model="advanceForm.ntpPort1" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="Server 1" v-if="paramKeys.includes('ntpServer1')" prop="ntpServer1" :rules="{
																	validator: function(rule, val, cb){
																		var reg = /^[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+$/;
																		if(val) {
																			if(!reg.test(val)) {
																				cb('You can enter characters such as a-z, A-Z, 0-9,. and -');
																			}else {
																				cb();
																			}
																		}else {
																			cb();
																		}
																	}
																}">
																	<el-input v-model="advanceForm.ntpServer1" maxlength="100" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="Port2" v-if="paramKeys.includes('ntpPort2')" prop="ntpPort2" :rules="{
																	validator: function(rule, val, cb){
																		var reg = /^(\d+\.\.){0,1}(\d+)$/;
																		if(val){
																			if(reg.test(val) && parseInt(val)>=1 && parseInt(val)<=65535) {
																				cb();
																			}else {
																				cb('Range: 1 - 65535');
																			}
																		}else{
																			cb();
																		}
																	}
																}">
																	<el-input v-model="advanceForm.ntpPort2" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="Server 2" v-if="paramKeys.includes('ntpServer2')" prop="ntpServer2" :rules="{
																	validator: function(rule, val, cb){
																		var reg = /^[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+$/;
																		if(val) {
																			if(!reg.test(val)) {
																				cb('You can enter characters such as a-z, A-Z, 0-9,. and -');
																			}else {
																				cb();
																			}
																		}else {
																			cb();
																		}
																	}
																}">
																	<el-input v-model="advanceForm.ntpServer2" maxlength="100" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="Port3" v-if="paramKeys.includes('ntpPort3')" prop="ntpPort3" :rules="{
																	validator: function(rule, val, cb){
																		var reg = /^(\d+\.\.){0,1}(\d+)$/;
																		if(val){
																			if(reg.test(val) && parseInt(val)>=1 && parseInt(val)<=65535) {
																				cb();
																			}else {
																				cb('Range: 1 - 65535');
																			}
																		}else{
																			cb();
																		}
																	}
																}">
																	<el-input v-model="advanceForm.ntpPort3" :disabled="isReadOnly"></el-input>
																</el-form-item>
																<el-form-item label="Server 3" v-if="paramKeys.includes('ntpServer3')" prop="ntpServer3" :rules="{
																	validator: function(rule, val, cb){
																		var reg = /^[a-zA-Z0-9-]+(\.[a-zA-Z0-9-]+)+$/;
																		if(val) {
																			if(!reg.test(val)) {
																				cb('You can enter characters such as a-z, A-Z, 0-9,. and -');
																			}else {
																				cb();
																			}
																		}else {
																			cb();
																		}
																	}
																}">
																	<el-input v-model="advanceForm.ntpServer3" maxlength="100" :disabled="isReadOnly"></el-input>
																</el-form-item>
															</div>
														</div>
													</el-collapse-item>
												</el-collapse>
							   					</el-form>
	                                        </div><!--自定义参数-->
											<el-collapse v-if="!isBLX">
												<el-collapse-item name="customized">
													<template slot='title'>
														<div style="display: flex;margin-left: 26px;">
															<span class='commonTextWeight14'><%=rb.getString("ZiDingYiCanShu")%></span>
															<div v-if="curType != 'view'" class='licenseImportBox' @click="customizedAddClick" style="margin: 10px 10px 0; position:relative; z-index: 9999">
																<i class='el-icon el-icon-plus importIcon'></i>
															</div>
														</div>
													</template>
													<div style='margin-right: 0px;' v-show="addOrEditForm.custParam.length > 0">
														<div style='width: 100%;' class='paramPoolWarpThree'>
															<div v-for="(item,index) in addOrEditForm.custParam" class="specialItemCls">
																<div style="display:flex;flex-wrap: wrap;font-size:12px">
																	<el-form-item label="ID" :prop="'custParam['+index+'].id'" class='validate-item' v-if="false"> 
																		<el-input v-model.trim='item.id' :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="Name" :prop="'custParam['+index+'].custParamName'" class='validate-item'> 
																		<el-input v-model.trim='item.custParamName' :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="Value" :prop="'custParam['+index+'].custParamValue'" class='validate-item'> 
																		<el-input v-model.trim='item.custParamValue' :disabled="isReadOnly"></el-input>
																	</el-form-item>
																	<el-form-item label="Trpath" :prop="'custParam['+index+'].custParamPath'"  class='validate-item'> 
																		<el-input v-model.trim='item.custParamPath' :disabled="isReadOnly" style="min-width: 250px;"></el-input>
																	</el-form-item>
																</div>
																<div class="specialItemDelIcon" v-if="curType != 'view'">
																	<span class="el-icon el-icon-circle-close" @click="customizedDelClick(item)" style="font-size: 12px;"></span>
																</div>
															</div>
															<el-form-item prop='custParam' style="display:none;" label-width="0px">
																<el-input v-model='addOrEditForm.custParam'></el-input>
															</el-form-item>
														</div>
													</div>
												</el-collapse-item>
											</el-collapse>
                                        </div>
                                    </el-tab-pane>
                                    <!--batch import-->
                                    <el-tab-pane label="<%=rb.getString("ZhiDingSheBeiJiHua")%>" name="paramTwo">
                                        <div class='paramTwoBox'>
                                            <el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='planConfigEnable' label-width='80px' class="enableCommon basicLeftLabel" style='margin-bottom: 16px;'>
                								<el-switch v-model="addOrEditForm.planConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="isReadOnly"></el-switch>
                                            </el-form-item>
											<div style='height: 300px;'>
												<el-ctable ref="ctablePlan" id='ctablePlan' :url="urlConfigPlan" :time="6" :query-params="params_plan" :row-key="'id'" class='commonBorderRadius commonBorder' style='margin-top: 10px; margin-bottom: 10px;'
													:height="height" pagination="true" rownumber="true">													
													<template slot="toolbar">
														<div class='commonFlex commonToolBarBox commonContent licenseBox'>
															<span class='commonTitle12' style='padding: 6px 20px;'><%=rb.getString("ZhiDingCanShuLieBiao")%></span>
															<div class='commonFlex'>
																<div class="queryGroup curImportQuery" style='margin-right: 6px;'>
																	<el-input v-model="params_plan_form.searchText" @keyup.enter.native="queryPlan" class='pairgrid-query' style='width: 220px;' placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-input>
																	<i @click="queryPlan" class="el-icon el-icon-common-search commonLeft10"></i>
																</div>
																<div v-show='curType !="view"' class='licenseImportBox' @click='specifyImportClick'><i class='el-icon el-icon-operation-import importIcon'></i></div>
																<div class='licenseImportBox' v-show='curProductName != "DXDF"' @click='specifyExportClick' style='margin-left: 0 !important;'><i class='el-icon el-icon-operation-export importIcon'></i></div>
															</div>
														</div>
													</template>
													<el-table-column width="100">
														<template slot-scope="scope">
															<div>
																<i v-show='curType != "view"' @click="specifyModifyClick(scope.row, event)" class="el-icon el-icon-operation-edit commonIcon"></i>
																<i v-show='curType != "view"' @click="specifyDeleteClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon" style='margin-left: 6px;'></i>
																<i  @click="specifyInfoClick(scope.row, event)" class="el-icon el-icon-operation-info commonIcon" style='margin-left: 6px;'></i>
															</div>
														</template>
													</el-table-column>
													<el-table-column v-if='false' label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" prop="platform"></el-table-column>
													<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number" min-width="200" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label="<%=rb.getString("HostName")%>" prop="host_name" width="200" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label="<%=rb.getString("ZhiChiPinDuan")%>" prop="bands_support" width="200" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label="<%=rb.getString("DaiKuan")%>" prop="band_width" :formatter="bandWidthFmt" width="200" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label="<%=rb.getString("ENBPinDian")%>" prop="frequency" :formatter="earfcnFmt" width="200" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label="<%=rb.getString("ZiZhenPeiBi")%>" prop="subframe_assignment" :formatter="sfassignmentFmt" width="200" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label="<%=rb.getString("GengXinRen")%>" prop="uploader"  width="140" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label="<%=rb.getString("GengXinShiJian")%>" prop="update_time" width="160" show-overflow-tooltip="true"></el-table-column>
												</el-ctable>												
								            </div>
                                        </div>
                                    	<!-- Regional 放在 Batch Import 中 -->
                                    	<div class='paramTwoBox commonBorderTop' style='margin-top: 20px;' v-if="curProductName == 'QAFA' || curProductName == 'QATA' || curProductName == 'DXDF'">
                                    		<span class='commonText14' style='margin: 10px 0 20px 0; display: inline-block;'><%=rb.getString("QuYuBuShu")%></span>
                                            <el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='selfAutoMapEnable' label-width='80px' class="enableCommon basicLeftLabel" style='margin-bottom: 16px;'>
                								<el-switch v-model="addOrEditForm.selfAutoMapEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6" :disabled="isReadOnly"></el-switch>
                                            </el-form-item>
											<div style='height: auto;'>
                                                <span class='commonTitle12'><%=rb.getString("QuYuBuShuTiShi")%></span>
												<div class='commonFlex licenseBox' style='position: relative; padding-top: 12px; flex-direction: row; justify-content: flex-start; width: 100%; flex-wrap: wrap; '>
													<span class='commonTitle12' style='padding: 6px 0; width: 130px;'><%=rb.getString("TACZiDongPeiZhi")%></span>
													<!-- MAP - 0, TAC - 1 -->
                                                    <el-radio-group v-model="addOrEditForm.autoConfigMode" style='margin-left: 10px; margin-top: 8px; display: flex;' @change='tacConfigChange' :disabled="isReadOnly">
                                                        <el-radio label="0">Location(key)-TAC</el-radio>
                                                        <el-radio label="1" style='margin-right: 10px;'>
                                                       		 eNB ID(key)-TAC 
                                                        	 <div v-show='addOrEditForm.autoConfigMode == "1"' style='margin-left: 10px; display: inline-block;'>
		                                                       	<el-checkbox v-model="addOrEditForm.defaultTac" true-label="1" false-label="0" :disabled="isReadOnly"></el-checkbox>
		                                                        <span class='commonLeft10'><%=rb.getString("TACPeiZhiTiShi")%></span> 
		                                                     </div>
                                                        </el-radio>
                                                        <el-radio label="2">SN(key)-TAC</el-radio>
                                                    </el-radio-group>
                                                    <div class='commonFlex' style='position: absolute; right: 22px;'>
                                                        <div v-show="addOrEditForm.autoConfigMode == '1' && curType !='view'" @click='enbIdTacAddClick' class='licenseImportBox'><i class='el-icon el-icon-plus importIcon'></i></div>
                                                        <div v-show="addOrEditForm.autoConfigMode == '2' && curType !='view'" @click='snAddClick' class='licenseImportBox'><i class='el-icon el-icon-plus importIcon'></i></div>
                                                        <div v-show="addOrEditForm.autoConfigMode == '0' && curType !='view'" @click='coordinateImportClick' class='licenseImportBox'><i class='el-icon el-icon-operation-import importIcon'></i></div>
                                                        <div v-show="addOrEditForm.autoConfigMode == '0' && curType !='view'" @click='coordinateClearClick' class='licenseImportBox' style='margin-left: 0 !important;'><i class='el-icon el-icon-operation-clear importIcon'></i></div>
                                                    </div>                                         
												</div>											
												<div v-show="addOrEditForm.autoConfigMode == '0'">
													<el-ctable id='ctableMap' ref="ctableMap" :url="urlMap" :time="6" :row-key="'id'" class='commonBorderRadius commonBorder' style='margin-top: 6px; margin-bottom: 10px;'
														height="260px" pagination="true" rownumber="true">
														<el-table-column width="70" v-if='curType != "view"'>
															<template slot-scope="scope">
																<div>
																	<i @click="coordinateModifyClick(scope.row, event)" class="el-icon el-icon-operation-edit commonIcon"></i>
																	<i @click="coordinateDeleteClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon" style='margin-left: 6px;'></i>
																</div>
															</template>
														</el-table-column>
														<el-table-column label="<%=rb.getString("SheBeiZuMingCheng")%>" prop="groupName"></el-table-column>
														<el-table-column label="TAC" prop="tac"></el-table-column>
														<el-table-column label="<%=rb.getString("ECIFanWei")%>" prop="eciRange"></el-table-column>														
													</el-ctable>
												</div>
												<div v-show="addOrEditForm.autoConfigMode == '1'">
													<!--  enb id tac -->
													<el-ctable id='enbIdTactable' ref="enbIdTactable" :url='enbIdUrl' :time="6" :row-key="'id'" class='commonBorderRadius commonBorder' style='margin-top: 6px;margin-bottom: 10px;'
														height="260px" pagination="true" rownumber="true">
														<el-table-column width="70" v-if='curType != "view"'>
															<template slot-scope="scope">
																<div>
 																	<i @click="enbIdModifyClick(scope.row, event)" class="el-icon el-icon-operation-edit commonIcon"></i>
 																	<i @click="enbIdDeleteClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon" style='margin-left: 6px;'></i>
																</div>
															</template>
														</el-table-column>
														<el-table-column label="TAC" prop="tac"></el-table-column>							
														<el-table-column label="eNB ID" prop="eciRange" :show-overflow-tooltip="true"></el-table-column>
													</el-ctable>
												</div>
												<div v-show="addOrEditForm.autoConfigMode == '2'">
													<el-ctable id='snctableMap' ref="snCtableMap" :url="snUrl" :time="6" :row-key="'id'" class='commonBorderRadius commonBorder' style='margin-top: 6px;margin-bottom: 10px;'
														height="260px" pagination="true" rownumber="true" :query-params="snQueryParam">
														<el-table-column width="90">
															<template slot-scope="scope">
																<div>
																	<i @click="snListClick(scope.row, event)" class="el-icon el-icon-devicelist commonIcon"></i>
																	<i v-if='curType != "view"' @click="snModifyClick(scope.row, event)" class="el-icon el-icon-operation-edit commonIcon" style='margin-left: 8px;'></i>
																	<i v-if='curType != "view"' @click="snDeleteClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon" style='margin-left: 6px;'></i>
																</div>
															</template>
														</el-table-column>
														<el-table-column label="<%=rb.getString("QuYu")%>" prop="groupName"></el-table-column>
														<el-table-column label="TAC" prop="tac"></el-table-column>
														<el-table-column label="ECI" prop="eciRange"></el-table-column>	
														<el-table-column label="MME IP" prop="mmeIp"></el-table-column>
													</el-ctable>
												</div>
								            </div>
                                        </div>
                                    </el-tab-pane>
                                </el-tabs>
                            </div>
			            </div>
			        </div>
                </el-form>   				
   			</div>
		    <div class='commonFlex commonContent footerBox commonBorderTop' v-show="curType !='view'">
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" size="mini" @click='enbAddOrUpdateSubmit' :disabled='saveBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='enbAddOrUpdateCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>            
   		<!--software upgrade: Add Version  -->
   		<div class='rightBox' style='position: relative;' v-show='addVersionShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("TianJianBanBen")%></span>
   				<span class='closeIconBox' @click='addVersionCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 10px 0 20px;'>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'><%=rb.getString("ShouDongHuoXuanZeBanBen")%></span></div>
   				<div class='originalVersionBox'>
   				   	<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '><%=rb.getString("ChuShiBanBen") %></span>
   				
   					<el-form ref="softwareAddVersionForm" :inline="true" label-position="top" :model="softwareAddVersionForm" :rules="softwareAddVersionRules" style="display: flex; flex-direction: column;overflow: hidden;">
	   					<el-form-item style='margin-bottom: 16px;'>
							<el-form-item prop='versionStr'>
								<el-input v-model='softwareAddVersionForm.versionStr'>
									 <i slot="append" class="el-icon el-icon-plus" @click="addVersionBtn"></i>
								</el-input>						
							</el-form-item>
							<div class='versionResultBox' v-show='softwareAddVersionForm.versionList.length > 0'>
								<el-form-item class='suffixItem' v-for='(domain,index) in softwareAddVersionForm.versionList'>
									<div class='form-suffix'>
										<span class='text'>{{domain}}</span>
										<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='removeVersion(domain)'></span>
									</div>
								</el-form-item>
								<el-form-item prop='itemTest'>
									<el-input v-model='softwareAddVersionForm.itemTest' v-show=false></el-input>
								</el-form-item>
							</div>
							<p class='ipErrorTip'>{{versionErrorMessage}}</p>
	                    </el-form-item>       
   					</el-form>
   				</div>
   				<div>
	                <span class='commonTitle12 commonDisplayBlock'><%=rb.getString("ChuShiBanBenLieBiao")%></span>	                  
					<el-ctable ref="softwareOriginalVersionTable" height='280px' class='commonBorderRadius commonBorder' style='margin-top: 8px;' :pagination="false" :rownumber="false" 
						:data="softwareOriginalVersionList" row-key="originalVersion" @selection-change='versionBatchSelect'>
						<el-table-column type="selection" :reserve-selection="true"></el-table-column>
						<el-table-column label="<%=rb.getString("ChuShiBanBen") %>" prop="originalVersion" show-overflow-tooltip="true"></el-table-column>
					</el-ctable>
	            </div>
	            <p class='ipErrorTip' style='margin-top: 2px;'>{{versionMessage}}</p>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" size="mini" @click='addVersionSubmit'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='addVersionCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div> 
   		<!--import license  -->
   		<div class='rightBox' style='position: relative;' v-show='licenseImportShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("DaoRu")%> License</span>
   				<span class='closeIconBox' @click='licenseImportCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'><%=rb.getString("DaoRuLicenseTiShi")%></span></div>
   				<div class='originalVersionBox'>
					<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '><%=rb.getString("WenJianMing")%>(<span class='commonTitle12'><%=rb.getString("LicenseGeShi")%></span>)</span>
   					<el-form label-position="top" ref="licenseForm" :model='licenseForm' :rules='importLicenseRules' v-loading="licenseLoading">     
	                  	<el-form-item label=" " prop="fileName">
							<el-upload 
				           		ref="moreUpload"
				            	:multiple="true"
				            	:on-success='moreCheckFile'
				            	:on-change="moreFileChange"
				            	:show-file-list=false
							    :action="licenseForm.uploadFileUrl"
							    :data="moreFileParams"
							    name="uploadFile"
							    :file-list="moreFileList" 
							    :http-request="moreFileRequest"
							    :auto-upload="false"
							    accept=".lic">
								<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
									<a slot="append" class="el-icon el-icon-operation-import importBox" @click="moreFileSelect"></a>
								</el-input>
								<a slot="trigger" ref="file_up"></a>
							</el-upload>	
	                    </el-form-item>      		                
		            </el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="licenseImportSubmit" :disabled='uploadBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="licenseImportCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
   		<!--Mpdel tyoe list start-->
   		<div class='rightBox320 commonBorder2 commonBackgroundWhite' style='position: relative;' v-show='modelTypeAddShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'>{{moduleTitle}}</span>
   				<span class='closeIconBox' @click='modelTypeAddCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='originalVersionBox width280'>   					
                  <el-form ref="moduleFm" :model="moduleForm" :rules="moduleRules" class="modifyMapForm addTacFormBox" label-position="top" >
						 <el-form-item label="<%=rb.getString("SheBeiXingHaoMing")%>" prop="module_type" style='margin-bottom: 22px;'>
							<el-input v-model='moduleVal' maxlength="20">
								<i slot="append" class="el-icon el-icon-plus" @click="addModuleType"></i>
							</el-input>			
							<div v-show='moduleForm.module_type.length > 0' class='versionResultBox' style='width: 280px;'>
								<el-form-item class='suffixItem' v-for='(domain,index) in moduleForm.module_type'>
									<div class='form-suffix' style='width: 92%; margin: 0 10px; padding: 0; width: 92%;'>
										<span class='text'>{{domain}}</span>
										<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click="removeModule(index)"></span>
									</div>
								</el-form-item>
							</div>
							<p class='ipErrorTip' style='margin-top: -6px; height: 12px;'>{{moduleErrorMessage}}</p>
						</el-form-item>
						<el-form-item label="<%=rb.getString("DaiKuan")%>" prop="band_width">
							<el-select v-model="moduleForm.band_width">
								<el-option label="5MHz" value="n25"></el-option>
								<el-option label="10MHz" value="n50"></el-option>
								<el-option label="15MHz" value="n75"></el-option>
								<el-option label="20MHz" value="n100"></el-option>
							</el-select>
						</el-form-item>						
						<el-form-item label="<%=rb.getString("ENBPinDian")%>" prop="frequency" class='validateRightItem'>
							<el-input v-model.trim="moduleForm.frequency">
								<template slot="append"><%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>:0~65535</template>
							</el-input>
						</el-form-item>						
					</el-form> 
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="modelTypeAddSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="modelTypeAddCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div> 
		<!-- Advance Setting Tree -->
		<div class='rightBox320' style='position: relative;' v-show='advanceTreeShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'>Add</span>
   				<span class='closeIconBox' @click='advanceSettingCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'>You can choose parameters to configure</span></div>
   				<div class='originalVersionBox'>   					
					<el-tree show-checkbox ref="advanceTree"
						node-key="id"
						:data="advanceTree"
						:check-on-click-node="true"
						:default-expanded-keys="['wan','ipsec','power','system']"
						:default-checked-keys="defaultRreeChecked">
					</el-tree>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop' style='width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100;'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="advanceTreeConfirm"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="advanceSettingCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
       	<!--batch import Start-->
        <!-- 导入 specifyImportShow,需要重新写逻辑-->
        <div class='rightBox' style='position: relative;' v-show='specifyImportShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox commonBorderBottom' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("DaoRu")%></span>
   				<span class='closeIconBox' @click='specifyImportCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;' >
   				<div class='originalVersionBox'>
					<el-form ref="importForm" :model="importForm" :rules="importRules" label-position="top">
						<el-form-item label="<%=rb.getString("DaoRuLeiXing")%>">
							<el-radio-group v-model="importForm.importType">
			                    <el-radio label="0" border><%=rb.getString("ZhuiJia")%></el-radio>
			                    <el-radio label="1" border><%=rb.getString("TiHuan")%></el-radio>
			                </el-radio-group>
						</el-form-item>
						<div v-loading="importSpecialLoading">
							<el-form-item label="<%=rb.getString("WenJian")%>" prop="filePath">
								<el-input :disabled="true" v-model="importForm.filePath">
									<i slot='append' @click="importFile" class='el-icon el-icon-operation-import'></i>
								</el-input>
							</el-form-item>
							<p class='commonTextNormal12'><%=rb.getString("DaoRuWenJianTiShi")%></p>
							<div style="position:relative;margin-top:10px;">
								<!-- R Series -->
								<div @click="exportTemp('intel')" v-if="curProductName == 'RTS' || curProductName == 'RTD'" class='commonTextNormal12' style='text-decoration:underline;cursor: pointer;'>
									<i class="el-icon el-icon-common-download commonIcon"></i><%=rb.getString("DaoChuMuBan")%>
								</div>
								<!-- Q Series -->
								<div @click="exportTemp('qa')" v-if="curProductName == 'QATA' || curProductName == 'QAFA' || curProductName == 'QAFB' || curProductName == 'QRTB' || curProductName == 'BAIBLQ' || curProductName == 'BLX' || curProductName == 'MLQ' || curProductName == 'CR-B4860' || curProductName == 'MLN'" class='commonTextNormal12' style='text-decoration:underline;cursor:pointer;'>
									<i class="el-icon el-icon-common-download commonIcon"></i><%=rb.getString("DaoChuMuBan")%>
								</div>
								<!-- NB Series -->
								<div @click="exportTemp('NBIOT')" v-if="curProductName == 'NBIOT'" class='commonTextNormal12' style='text-decoration:underline;cursor:pointer;'>
									<i class="el-icon el-icon-common-download commonIcon"></i><%=rb.getString("DaoChuMuBan")%>
								</div>							
							</div>
						</div>
					</el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="specifyImportSubmit" :disabled='importLoading'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="specifyImportCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
        <!--修改 specifyModifyShow -->
        <div class='specifyModifyRightBox infoSpecifiedDevice' style='position: relative;' v-show='specifyModifyShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox commonBorderBottom' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14' style='font-size: 16px;'><%=rb.getString("XiuGai")%></span>
   				<span class='closeIconBox' @click='specifyModifyCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent modifyForm' style='margin: 0 20px; height: calc(100% - 86px); overflow-y: scroll;'>
   				<div style='margin-bottom: 40px;'>
   					<el-form ref="basicForm" :model="basicForm" :rules="basicRules" label-position="top">
   						<el-collapse v-model="activeNames">
							<el-collapse-item name="basic">
								<template slot='title'>
									<p style="display:inline-block;margin-left: 26px;">
										<span class='commonTextWeight12'>{{modifyBasicTitle}}</span>
									</p>
									<span @click='showCell2Param' v-show='curType != "view" && (curProductName == "CR-B4860" || curProductName == "MLN")' class='licenseImportBox' style="display: inline-block;"><i class='el-icon el-icon-plus importIcon' style='margin: 0;'></i></span>
								</template>
								<div class='commonBorderBottom paramsBasicItemWarp'>
									<el-form-item label="<%=rb.getString("ZhiChiPinDuan")%>" prop="bands_support">
										<el-input v-model="basicForm.bands_support"></el-input>
									</el-form-item>
									<p class='item_label' v-if=false><%=rb.getString("ZhiChiPinDuan")%> : <span style='color:#000'>{{basicForm.bands_support}}</span></p>
									<el-form-item label='<%=rb.getString("DaiKuan")%>' prop="band_width" v-show='curProductName == "DXDF"'>
										<el-select v-model="basicForm.band_width">
											<el-option label="6" value="6"></el-option>
											<el-option label="15" value="15"></el-option>
											<el-option label="25" value="25"></el-option>
											<el-option label="50" value="50"></el-option>
											<el-option label="75" value="75"></el-option>
											<el-option label="100" value="100"></el-option>
										</el-select>	
									</el-form-item>
									<el-form-item label='<%=rb.getString("DaiKuan")%>' prop="band_width" v-show='shownbi && curProductName != "DXDF"'>
										<el-select v-model="basicForm.band_width" :disabled="sasDisabled">
											<el-option label="5MHz" value="n25"></el-option>
											<el-option label="10MHz" value="n50"></el-option>
											<el-option label="15MHz" value="n75"></el-option>
											<el-option label="20MHz" value="n100"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ENBPinDian")%>' prop="frequency" v-show='curProductName == "DXDF"'>
										<el-input v-model="basicForm.frequency"></el-input>
									</el-form-item>
									<el-form-item :label="frequencyLabel" prop="frequency" v-show='curProductName != "DXDF"'>
										<el-input v-model="basicForm.frequency" @blur="changeToFrequency('dl')" @focus="changeToEarfcn('dl')" :disabled="sasDisabled"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ZiZhenPeiBi")%>' prop="subframe_assignment" v-show='showBasic && curProductName != "QAFA" && curProductName != "QAFB" && curProductName != "DXDF"'>
										<el-select v-model="basicForm.subframe_assignment">
											<el-option label='<%=rb.getString("QingXuanZe")%>' value=""></el-option>
											<el-option label="0(DL:UL = 1:3)" value="0" v-show='curProductName == "RTS" || curProductName == "RTD"'></el-option>
											<el-option label="1(DL:UL = 2:2)" value="1"></el-option>
											<el-option label="2(DL:UL = 3:1)" value="2"></el-option>
											<el-option label='6(DL:UL = 3:5)' value='6' v-show='curProductName == "QRTB" || curProductName == "BAIBLQ" || curProductName == "BLX" || curProductName == "MLQ" || curProductName == "CR-B4860" || curProductName == "MLN"'></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("TeShuZiZhenPeiBi")%>' prop="special_subframe_patterns" v-show='showBasic && curProductName != "QAFA" && curProductName != "QAFB"  && curProductName != "DXDF"'>
										<el-select v-model="basicForm.special_subframe_patterns">
											<el-option label='<%=rb.getString("QingXuanZe")%>' value=""></el-option>
											<el-option label="5" value="5"></el-option>
											<el-option label="7" value="7"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("PLMN")%>' prop="plmn_id">
										<el-input v-model="basicForm.plmn_id"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("TAC")%>' prop="tac">
										<el-input v-model="basicForm.tac"></el-input>
									</el-form-item>
									<el-form-item :label="eciLable" prop="cell_identity">
										<el-input v-model="basicForm.cell_identity"></el-input>
									</el-form-item>
									<el-form-item label="PCI" prop="phycellid">
										<el-input v-model="basicForm.phycellid"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("CPETxPower")%>' prop="dxdfTxPower" v-show='curProductName == "DXDF"'>
										<el-select v-model="basicForm.dxdfTxPower" filterable multiple collapse-tags>
											<el-option v-for="item in dxdfPowerList" :label="item.label" :value="item.value"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("CPETxPower")%>' prop="txPower" key="txPower" v-if="mapFlag && curProductName != 'DXDF' && (curProductName =='QAFA' || curProductName =='QATA')" key="txPower">
										<el-select v-model="basicForm.txPower" filterable>
											<el-option v-for="item in powerList" :key="item.value" :label="item.label" :value="item.value"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("GenXuLieSuoYin")%>' prop="root_sequence_index" v-show='shownbi && curProductName != "DXDF"'>
										<el-input v-model="basicForm.root_sequence_index"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ZaiBoLeiXing")%>' prop="carrier_mode" v-if='showMode && curProductName != "DXDF"'>
										<el-select v-model="basicForm.carrier_mode">
											<el-option label='<%=rb.getString("DanZaiBo")%>' value="1"></el-option>
											<el-option label='<%=rb.getString("ShuangZaiBo")%>' value="2"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ShiQuSheZhi")%>' prop="timeZoneUtc" v-if="mapFlag && (curProductName =='QAFA' || curProductName =='QATA') && curProductName != 'DXDF'" key="timeZoneUtc">
										<el-select v-model="basicForm.timeZoneUtc" filterable>
											<el-option v-for="item in timeZoneList" :key="item.value" :label="item.label" :value="item.value"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("HaloBKaiGuan")%>' prop="halob_enable" v-if='curProductName != "DXDF"'>
										<el-select v-model="basicForm.halob_enable" @change="changeHalob">
											<el-option label='<%=rb.getString("HalobKaiQi")%>' value="1"></el-option>
											<el-option label='<%=rb.getString("HalobGuanBi")%>' value="0"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item v-if="showMME && curProductName != 'DXDF'" label="MME">
										<el-input v-model='basicForm.mme'>
											 <i slot="append" class="el-icon el-icon-plus" @click="addMME('')"></i>
										</el-input>	
										<div class='versionResultBox' v-show='basicForm.mmeGroup.length > 0' style='width: 200px;'>
											<div class='suffixItem' v-for='(domain,index) in basicForm.mmeGroup'>
												<div class='form-suffix' style='width: 170px; margin: 3px 10px 0;'>
													<span class='text'>{{domain}}</span>
													<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='removeMME(domain, "")'></span>
												</div>
											</div>
										</div>
										<p class='ipErrorTip'>{{errorMsg}}</p>
									</el-form-item>
									<el-form-item prop="mmeStr" style="margin-right: -1px;margin-bottom: 0;">
										<el-input v-model="basicForm.mmeStr" v-show=false></el-input>
									</el-form-item>	
								</div>
							</el-collapse-item>
							<el-collapse-item name="cell" v-if='(curProductName == "CR-B4860" || curProductName == "MLN") && showCell2 == true'>
								<template slot='title'>
									<p style="display:inline-block;margin-left: 26px;">
										<span class='commonTextWeight12'><%=rb.getString("XiaoQuCanShu")%> 2</span>
									</p>
								</template>
								<div class='commonBorderBottom paramsBasicItemWarp'>								
									<el-form-item label='<%=rb.getString("PLMN")%>' prop="cellPlmnId">
										<el-input v-model="basicForm.cellPlmnId"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("DaiKuan")%>' prop="cellBandwidth">
										<el-select v-model="basicForm.cellBandwidth" :disabled="sasDisabled">
											<el-option label="5MHz" value="n25"></el-option>
											<el-option label="10MHz" value="n50"></el-option>
											<el-option label="15MHz" value="n75"></el-option>
											<el-option label="20MHz" value="n100"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item label='<%=rb.getString("ENBPinDian")%>' prop="cellFrequency">
										<el-input v-model="basicForm.cellFrequency" @blur="cellChangeToFrequency('dl')" @focus="cellChangeToEarfcn('dl')" :disabled="sasDisabled"></el-input>
									</el-form-item>
									<el-form-item label="ECI (ECI=eNB_ID*256+Cell_ID)" prop="cellEci">
										<el-input v-model="basicForm.cellEci"></el-input>
									</el-form-item>
									<el-form-item label="PCI" prop="cellPhycellid">
										<el-input v-model="basicForm.cellPhycellid"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("GenXuLieSuoYin")%>' prop="cellRootIndex">
										<el-input v-model="basicForm.cellRootIndex"></el-input>
									</el-form-item>
									<el-form-item label="MME" v-if="showMME">
										<el-input v-model='basicForm.cellMme'>
											 <i slot="append" class="el-icon el-icon-plus" @click="addMME('cell')"></i>
										</el-input>	
										<div class='versionResultBox' v-show='basicForm.cellMmeGroup.length > 0' style='width: 200px;'>
											<div class='suffixItem' v-for='(domain,index) in basicForm.cellMmeGroup'>
												<div class='form-suffix' style='width: 170px; margin: 3px 10px 0;'>
													<span class='text'>{{domain}}</span>
													<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='removeMME(domain,"cell")'></span>
												</div>
											</div>
										</div>
										<p class='ipErrorTip'>{{cellErrorMsg}}</p>
									</el-form-item>
									<el-form-item prop="cellMmeStr" style="margin-right: -1px;margin-bottom: 0;">
										<el-input v-model="basicForm.cellMmeStr" v-show=false></el-input>
									</el-form-item>
								</div>
							</el-collapse-item>
						</el-collapse>
   					</el-form>
   					<!-- settings -->
   					<el-form ref="settingForm" :model="settingForm" :rules="settingRules" label-position="top" v-if="showSetting && curProductName != 'DXDF'">
						<el-collapse>
							<!-- setting -->
							<el-collapse-item> 
								<template slot='title'>
									<el-form-item style="display:inline-block;margin-left:26px;margin-top:12px;" prop="ipsec_switch">
										<span class='commonTextWeight12'><%=rb.getString("SheZhi")%></span>
										<el-checkbox v-model="settingForm.ipsec_switch" label=" "></el-checkbox>
									</el-form-item>
								</template>
								<div>
									<el-form-item prop="ipsec_enable" label="<%=rb.getString("IpsecKaiGuan")%>" style='margin-bottom: 10px; '>
						                <el-radio-group v-model="settingForm.ipsec_enable">
						                    <el-radio label="1" border><%=rb.getString("QiYong")%></el-radio>
						                    <el-radio label="0" border><%=rb.getString("JinYong")%></el-radio>
						                </el-radio-group>
						            </el-form-item>
									<el-form-item v-if="showItem" label="<%=rb.getString("IKEXieShangMuDiDuanKou")%>" prop="ipsec_rightikeport" class="inputItem">
										<el-select v-model="settingForm.ipsec_rightikeport">
											<el-option label="500" value="500"></el-option>
											<el-option label="4500" value="4500"></el-option>
										</el-select>
									</el-form-item>
									<el-form-item v-if="showItem" label="Left Interface" class="inputItem" prop="left_interface">
										<el-select v-model="settingForm.left_interface">
											<el-option label="none" value="none"></el-option>
											<el-option label="WAN(eth2)" value="WAN"></el-option>
											<el-option label="PPPOE(pppoe-wan)" value="PPPOE"></el-option>
										</el-select>
									</el-form-item>
									<div  style="padding-bottom: 10px;">
										<div style="display:flex;width:95%">
											<label style="flex:1 1 auto;line-height:30px;">Ipsec Tunnel Table</label>
											<span v-show='curType !="view"' @click="addIpsec" class='el-icon el-icon-circle-add' style='font-size:20px;'></span>
										</div>
										<div style="height:200px;max-height:200px;width:95%;border: 1px solid #E9EDF9;">
											<el-ctable ref="ctableIpsec" :data="settingForm.ipsecList" :pagination="false" :rownumber="false">
												<el-table-column label="" width="30" prop="" class-name="no-text-tips">
													<template slot-scope="scope">
														<div class="el-icon el-icon-operation-more" @click="optClickIpsec(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
													</template>
												</el-table-column>
												<el-table-column label="Tunnel ID" prop="IPSEC_INDEX"></el-table-column>
												<el-table-column label="<%=rb.getString("TunnelKaiGuan")%>" prop="TUNNEL_ENABLE" width='110' :formatter="tunnelEnableFmt" show-overflow-tooltip="true"></el-table-column>
												<el-table-column label="<%=rb.getString("TunnelMingCheng")%>" prop="TUNNEL_NAME" width='110' show-overflow-tooltip="true"></el-table-column>
												<el-table-column label="<%=rb.getString("TunnelWangGuan")%>" prop="TUNNEL_GATEWAY" width='120' show-overflow-tooltip="true"></el-table-column>
											</el-ctable>
 											<el-cmenu ref="menu_ipsec" :data="menus_ipsec" @click="clickMenuIpsec"></el-cmenu>
										</div>
										<el-form-item prop="ipsecLength" style="margin-bottom:0px;width:300px;" class="inputItem">
											<el-input v-model="settingForm.ipsecLength" v-show=false></el-input>
										</el-form-item>
									</div>
								</div>
							</el-collapse-item>
						</el-collapse>
					</el-form>  
					<!-- wan Config -->
   					<el-form v-if="curProductName == 'CR-B4860' || curProductName == 'BAIBLQ'" ref="wanConfigForm" :model="wanConfigForm" :rules="wanConfigRules" label-position="top">
						<el-collapse>
							<el-collapse-item> 
								<template slot='title'>
									<el-form-item style="display:inline-block;margin-left:26px;margin-top:12px;" prop="wanSendEnable">
										<span class='commonTextWeight12'>WAN Config</span>
										<el-checkbox v-model="wanConfigForm.wanSendEnable" true-label="1" false-label="0"></el-checkbox>
									</el-form-item>
								</template>
								<div class='paramsBasicItemWarp'>
									<div v-for='(item, index) in wanConfigForm.wanBindingList' :key="index" style='margin-bottom: 15px; padding: 10px 0 0 20px; border: 1px solid #D5DCEC; border-radius: 2px; background: #FCFCFC;box-sizing: border-box;'>
										<div style="display: flex; justify-content: flex-start;">
											<span class='tieleTipCircle'></span>
											<span class='commonGeneralBold12' style='padding: 6px 0 10px;'>{{wanNameList[index]}}</span>		
										</div> 
										<el-form-item label='<%=rb.getString("IPDiZhi")%>' :prop="'wanBindingList.' + index + '.ipAddress'">
											<el-input v-model="item.ipAddress"></el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("ZiWangYanMa")%>' :prop="'wanBindingList.' + index + '.netmask'">
											<el-input v-model="item.netmask"></el-input>
										</el-form-item>
										<el-form-item label='<%=rb.getString("WangGuan")%>' :prop="'wanBindingList.' + index + '.gateway'">
											<el-input v-model="item.gateway"></el-input>
										</el-form-item>
										<el-form-item label='VLAN ID' :prop="'wanBindingList.' + index + '.vlanId'">
											<el-input v-model="item.vlanId"></el-input>
										</el-form-item>
										<el-form-item :label='wanBindingLabelList[index]' :prop="'wanBindingList.' + index + '.wanBinding'">
											<el-input v-model="item.wanBinding"></el-input>
										</el-form-item>										
									</div>
								</div>
							</el-collapse-item>

						</el-collapse>
					</el-form> 					
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="specifyModifySubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="specifyModifyCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
        <!--详情 specifyInfoShow-->
        <div class='rightBox infoSpecifiedDevice' style='position: relative;' v-show='specifyInfoShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14' style='font-size: 16px;'>SN:{{specifySN}}</span>
   				<span class='closeIconBox' @click='specifyInfoCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 10px 0 20px;'>
                <span class=' commonSize14'>Cell Name:{{cellName}}</span>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'>Parameters configuration planning information.</span></div>   				
   				<!-- 接口读取数据 -->
   				<div style='height: calc( 100% - 110px); overflow-y: scroll;'>
   					<el-collapse v-model="infoActiveName">
                        <!-- Basic -->
                        <el-collapse-item name="basic">
                            <template slot='title'>
                                <p style="display:inline-block;margin-left: 26px;">
                                    <span class='commonTextWeight12'>{{modifyBasicTitle}}</span>
                                </p>
                            </template>
							<div v-if='curProductName == "DXDF"'>
								<div class='commonFlex commonContent commonText' style='padding-top: 0;'>
                                    <span class='commonTitle12'><%=rb.getString("ZhiChiPinDuan")%></span>
                                    <span class='commonText12 marginRight10'>{{bands_support}}</span>
                                </div>
								<div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'><%=rb.getString("DaiKuan")%></span>
                                    <span class='commonText12 marginRight10'>{{band_width}}</span>
                                </div>
								<div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'><%=rb.getString("ENBPinDian")%></span>
                                    <span class='commonText12 marginRight10'>{{frequency}}</span>
                                </div>
								<div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'><%=rb.getString("TAC")%></span>
                                    <span class='commonText12 marginRight10'>{{tac}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'>CELL ID</span>
                                    <span class='commonText12 marginRight10'>{{cell_identity}}</span>
                                </div>
								<div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'>PCI</span>
                                    <span class='commonText12 marginRight10'>{{phycellid}}</span>
                                </div>
								<div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'><%=rb.getString("PLMN")%></span>
                                    <span class='commonText12 marginRight10'>{{plmn_id}}</span>
                                </div>
								<div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'><%=rb.getString("CPETxPower")%></span>
                                    <span class='commonText12 marginRight10'>{{txPower}}</span>
                                </div> 
							</div>
                            <div v-else class='commonBorderBottom'>
                                <div class='commonFlex commonContent commonText' style='padding-top: 0;'>
                                    <span class='commonTitle12'><%=rb.getString("ZhiChiPinDuan")%></span>
                                    <span class='commonText12 marginRight10'>{{bands_support}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText' v-show='ipsecInfoShownbi'>
                                    <span class='commonTitle12'><%=rb.getString("DaiKuan")%></span>
                                    <span class='commonText12 marginRight10'>{{band_width}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'>{{frequencyLabel}}</span>
                                    <span class='commonText12 marginRight10'>{{frequency}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText' v-show='ipsecInfoShowBasic && curProductName != "QAFA" && curProductName != "QAFB"'>
                                    <span class='commonTitle12'><%=rb.getString("ZiZhenPeiBi")%></span>
                                    <span class='commonText12 marginRight10'>{{subframe_assignment}}</span>
                                </div>
                                 <div class='commonFlex commonContent commonText' v-show='ipsecInfoShowBasic && curProductName != "QAFA" && curProductName != "QAFB"'>
                                    <span class='commonTitle12'><%=rb.getString("TeShuZiZhenPeiBi")%></span>
                                    <span class='commonText12 marginRight10'>{{special_subframe_patterns}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'><%=rb.getString("PLMN")%></span>
                                    <span class='commonText12 marginRight10'>{{plmn_id}}</span>
                                </div>                           
                                <div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'><%=rb.getString("TAC")%></span>
                                    <span class='commonText12 marginRight10'>{{tac}}</span>
                                </div>
                                 <div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'>ECI (ECI=eNB_ID*256+Cell_ID)</span>
                                    <span class='commonText12 marginRight10'>{{cell_identity}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'>PCI</span>
                                    <span class='commonText12 marginRight10'>{{phycellid}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText' v-show='ipsecInfoShownbi'>
                                    <span class='commonTitle12'><%=rb.getString("GenXuLieSuoYin")%></span>
                                    <span class='commonText12 marginRight10'>{{root_sequence_index}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText' v-show='ipsecInfoShowMode'>
                                    <span class='commonTitle12'><%=rb.getString("ZaiBoLeiXing")%></span>
                                    <span class='commonText12 marginRight10'>{{carrier_mode}}</span>
                                </div>                              
                                <div class='commonFlex commonContent commonText' v-show="ipsecInfoMapFlag && (curProductName =='QAFA' || curProductName =='QATA')">
                                    <span class='commonTitle12'><%=rb.getString("ShiQuSheZhi")%></span>
                                    <span class='commonText12 marginRight10'>{{timeZoneUtc}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText' v-show="ipsecInfoMapFlag && (curProductName =='QAFA' || curProductName =='QATA')">
                                    <span class='commonTitle12'><%=rb.getString("CPETxPower")%></span>
                                    <span class='commonText12 marginRight10'>{{txPower}}</span>
                                </div> 
                                <div class='commonFlex commonContent commonText'>
                                    <span class='commonTitle12'><%=rb.getString("HaloBKaiGuan")%></span>
                                    <span class='commonText12 marginRight10'>{{halob_enable}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText' style='padding-bottom: 10px;' v-show='ipsecInfoShowMME'>
                                    <span class='commonTitle12'>MME</span>
                                    <span class='commonText12' style='margin-left: 20px; margin-right: 10px; max-width: 170px; word-break: break-all; word-wrap: break-word;'>{{mmeStr}}</span>
                                </div>                            
                            </div>
                        </el-collapse-item>
						<el-collapse-item name="cell" v-if='showInfoCell2 && ( curProductName == "CR-B4860" || curProductName == "MLN" )'>
                            <template slot='title'>
                                <p style="display:inline-block;margin-left: 26px;">
                                    <span class='commonTextWeight12'><%=rb.getString("XiaoQuCanShu")%>2</span>
                                </p>
                            </template>

							<div class='commonFlex commonContent commonText'>
								<span class='commonTitle12'><%=rb.getString("PLMN")%></span>
								<span class='commonText12 marginRight10'>{{cellPlmnId}}</span>
							</div>
							<div class='commonFlex commonContent commonText'>
								<span class='commonTitle12'><%=rb.getString("DaiKuan")%></span>
								<span class='commonText12 marginRight10'>{{cellBandwidth}}</span>
							</div>
							<div class='commonFlex commonContent commonText'>
								<span class='commonTitle12'><%=rb.getString("ENBPinDian")%></span>
								<span class='commonText12 marginRight10'>{{cellFrequency}}</span>
							</div>
							<div class='commonFlex commonContent commonText'>
								<span class='commonTitle12'>ECI (ECI=eNB_ID*256+Cell_ID)</span>
								<span class='commonText12 marginRight10'>{{cellEci}}</span>
							</div>
							<div class='commonFlex commonContent commonText'>
								<span class='commonTitle12'>PCI</span>
								<span class='commonText12 marginRight10'>{{cellPhycellid}}</span>
							</div>
							<div class='commonFlex commonContent commonText'>
								<span class='commonTitle12'><%=rb.getString("GenXuLieSuoYin")%></span>
								<span class='commonText12 marginRight10'>{{cellRootIndex}}</span>
							</div>
							<div class='commonFlex commonContent commonText' style='padding-bottom: 10px;' v-show='ipsecInfoShowMME'>
								<span class='commonTitle12'>MME</span>
								<span class='commonText12' style='margin-left: 20px; margin-right: 10px; max-width: 170px; word-break: break-all; word-wrap: break-word;'>{{cellInfoMme}}</span>
							</div>	
						</el-collapse-item>
   					</el-collapse>
                    <el-collapse v-if="ipsecInfoShowSetting">
                        <!-- Settings -->
                        <el-collapse-item>
                           <template slot='title'>
                                <p style="display:inline-block;margin-left: 26px;">
                                    <span class='commonTextWeight12'><%=rb.getString("SheZhi")%></span>
                                    <el-checkbox v-model="ipsec_switch" label=" " disabled></el-checkbox>
                                </p>
                            </template>
                            <div class='commonBorderBottom'>
                                <div class='commonFlex commonContent commonText' style='padding-top: 0;'>
                                    <span class='commonTitle12'><%=rb.getString("IpsecKaiGuan")%></span>
                                    <span class='commonText12 marginRight10'>
	                                    <el-switch v-model="ipsec_enable" style="height: 18px;" disabled		        							
		        							:active-value="'1'" 
		        							:inactive-value="'0'" 
		        							active-color="#4D84FF" 
		        							inactive-color="#CFCFCF">	
	        							</el-switch>
                                    </span>
                                </div>
                                <div class='commonFlex commonContent commonText' style='padding-top: 0;' v-show="ipsecInfoShowItem">
                                    <span class='commonTitle12'><%=rb.getString("IKEXieShangMuDiDuanKou")%></span>
                                    <span class='commonText12 marginRight10'>{{ipsec_rightikeport}}</span>
                                </div>
                                <div class='commonFlex commonContent commonText' style='padding-top: 0;' v-show="ipsecInfoShowItem">
                                    <span class='commonTitle12'>Left Interface</span>
                                    <span class='commonText12 marginRight10'>{{left_interface}}</span>
                                </div>
                                <div class='commonText' style='margin: 0 10px 20px 0;'>
                                    <span class='commonTitle12'>Ipsec Tunnel Table</span>
                                   	<div class='commonBorder' style='padding-top: 5px; margin-bottom: 5px;' v-for='item in ipsecList'>
                                   		<div class='commonFlex commonContent commonText1' style='padding: 6px 10px 3px;'>
		                                    <span class='commonTitle12'>Tunnel ID</span>
		                                    <span class='commonText12'>{{item.IPSEC_INDEX}}</span>
		                                </div>
		                                <div class='commonFlex commonContent commonText1' style='padding: 3px 10px;'>
		                                    <span class='commonTitle12'><%=rb.getString("TunnelKaiGuan")%></span>
		                                    <span class='commonText12'>
			                                    <el-switch v-model="item.TUNNEL_ENABLE" style="height: 18px;" disabled	        							
				        							:active-value="'1'" 
				        							:inactive-value="'0'" 
				        							active-color="#4D84FF" 
				        							inactive-color="#CFCFCF">	
			        							</el-switch>
		                                    </span>		                                    
		                                </div>
		                                <div class='commonFlex commonContent commonText1' style='padding: 3px 10px;'>
		                                    <span class='commonTitle12'><%=rb.getString("TunnelMingCheng")%></span>
		                                    <span class='commonText12'>{{item.TUNNEL_NAME}}</span>
		                                </div>
		                                <div class='commonFlex commonContent commonText1' style='padding: 3px 10px 6px;'>
		                                    <span class='commonTitle12'><%=rb.getString("TunnelWangGuan")%></span>
		                                    <span class='commonText12'>{{item.TUNNEL_GATEWAY}}</span>
		                                </div>
                                   	</div>
                                </div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
					<el-collapse v-if="(curProductName == 'CR-B4860' || curProductName == 'BAIBLQ') && infoWanBindingList.length > 0">
						<el-collapse-item> 
							<template slot='title'>
								<p style="display:inline-block;margin-left: 26px;">
                                    <span class='commonTextWeight12'>WAN Config</span>
                                    <el-checkbox v-model="infoWanSendEnable" true-label="1" false-label="0" disabled></el-checkbox>
                                </p>
							</template>
							<div v-for='(item, index) in infoWanBindingList' :key="index" style='margin-bottom: 5px; padding: 5px 0 0 10px; border: 1px solid #E9E9E9; border-radius: 2px; background: #FFFFFF;box-sizing: border-box;'>
								<div style="display: flex; justify-content: flex-start;">
									<span class='tieleTipCircle'></span>
									<span class='commonGeneralBold12' style='padding: 6px 0 10px;'>{{wanNameList[index]}}</span>		
								</div> 
								<div class='commonFlex commonContent commonText1' style='padding: 0 10px 3px;'>
									<span class='commonTitle12'><%=rb.getString("IPDiZhi")%></span>
									<span class='commonText12'>{{item.ipAddress}}</span>
								</div>
								<div class='commonFlex commonContent commonText1' style='padding: 3px 10px;'>
									<span class='commonTitle12'><%=rb.getString("ZiWangYanMa")%></span>
									<span class='commonText12'>{{item.netmask}}</span>
								</div>
								<div class='commonFlex commonContent commonText1' style='padding: 3px 10px;'>
									<span class='commonTitle12'><%=rb.getString("WangGuan")%></span>
									<span class='commonText12'>{{item.gateway}}</span>
								</div>
								<div class='commonFlex commonContent commonText1' style='padding: 3px 10px;'>
									<span class='commonTitle12'>VLAN ID</span>
									<span class='commonText12'>{{item.vlanId}}</span>
								</div>
								<div class='commonFlex commonContent commonText1' style='padding: 3px 10px 6px;'>
									<span class='commonTitle12'>{{wanBindingLabelList[index]}}</span>
									<span class='commonText12' style='display: inline-block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 190px;' :title='item.wanBinding'>{{item.wanBinding}}</span>
								</div>									
							</div>
						</el-collapse-item>
					</el-collapse>
   				</div>
   			</div>
   		</div>
		<!-- batch import End-->
		
		<!--Region deploying start  -->
        <!--import Tac  -->
   		<div class='rightBox' style='position: relative;' v-show='coordinateTacImportShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("DaoRu")%></span>
   				<span class='closeIconBox' @click='coordinateImportCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;' >
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'>Import .KML file by append method.</span></div>
   				<div class='originalVersionBox'>
					<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '>TAC-POLYGON File(<span class='commonTitle12'>Only .KML is supported</span>)</span>
   					<el-form ref="mapForm" :model="mapForm" :rules="mapRule" label-position="top" v-loading="mapLoading"  style='height: 60%;'>
						<el-upload
							ref="upload_polygon"
							:before-upload='beforeUploadPolygon' 
							:show-file-list=false 
							:auto-upload="false"
							:on-change="fileChangePolygon"  
							action=""
							accept=".kml">
							<el-form-item prop="file_polygon">							
								<el-input :readonly="true" :value='mapForm.file_polygon' placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
									<a slot="append" class="el-icon el-icon-operation-import importBox" @click="selectPolFile"></a>
								</el-input>
							</el-form-item>
							<a slot="trigger" ref="file_polygon"></a>
						</el-upload>						
					</el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="coordinateImportSubmit" :disabled='mapSubmitDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="coordinateImportCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
        <!--import end -->
        <!--Modify KML file Parameters start-->
   		<div class='rightBox' style='position: relative;' v-show='coordinateTacModifyShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'><%=rb.getString("XiuGaiKMLWenJianCanShu")%></span>
   				<span class='closeIconBox' @click='coordinateModifyCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'><%=rb.getString("XiuGaiKMLWenJianZhongDeCanShu")%></span></div>
   				<div class='originalVersionBox'>   					
   					<el-form ref="modifyMapForm" :model="modifyMapForm" :rules="editMapRule" label-position="top" label-width="120px" class="modifyMapForm">
                        <el-form-item label="<%=rb.getString("SheBeiZuMingCheng")%>" prop="groupName">
                            <el-input v-model="modifyMapForm.groupName"></el-input>
                        </el-form-item>
                        <el-form-item label="TAC" prop="tac">
                            <el-input v-model="modifyMapForm.tac"></el-input>
                        </el-form-item>
                        <el-form-item label="<%=rb.getString("ECIFanWei")%>" prop="idRange">
                            <el-input v-model="idRangeStart" style="width:145px;"></el-input> 一
                            <el-input v-model="idRangeEnd" style="width:145px;"></el-input>
                        </el-form-item>
                    </el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="coordinateModifySubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="coordinateModifyCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
        <!--add eNB ID-tAC start-->
   		<div class='rightBox320' style='position: relative;' v-show='enbIdTacModifyShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'>{{AddModifyTacTitle}}</span>
   				<span class='closeIconBox' @click='enbIdTacAddCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'><%=rb.getString("TianJiaENBIDTACTiShi")%></span></div>
   				<div class='originalVersionBox'>   					
   					<el-form ref="addTacForm" :model="addTacForm" :rules="addTacRule" label-position="top" label-width="80px" class="modifyMapForm addTacFormBox">			
                        <el-form-item label="TAC" prop="tac">
                            <el-input v-model="addTacForm.tac" :disabled='enbIdTacDisabled' style='width: 270px;'></el-input>
                        </el-form-item>
                        <el-form-item label="eNB ID" prop="eciRange" class='enbIdItem' style='margin-bottom: 2px;'>
                            <el-input type="textarea" v-model="addTacForm.eciRange"></el-input>
                        </el-form-item>
                        <span class='enbIdTip commonTitle12' v-show='enbIdTip'><%=rb.getString("eNBIDShuRuTiShi")%></span>
                    </el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="enbIdTacAddSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="enbIdTacAddCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div> 
        <!--add sn-->
        <div class='rightBox320' style='position: relative;' v-show='snAddShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'>{{AddModifySnTitle}}</span>
   				<span class='closeIconBox' @click='snAddCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='originalVersionBox'>   					
   					<el-form ref="addSnForm" :model="addSnForm" :rules="addSnRule" label-position="top" label-width="80px" class="modifyMapForm addTacFormBox">			
                        <el-form-item label="<%=rb.getString("QuYu")%>" prop="groupName">
                            <el-input v-model="addSnForm.groupName" :disabled='snNameDisabled' style='width: 270px;'></el-input>
                        </el-form-item>
                        <el-form-item label="TAC" prop="tac">
                            <el-input v-model="addSnForm.tac" style='width: 270px;'></el-input>
                        </el-form-item>
                        <el-form-item label="ECI" prop="eci" class='enbIdItem' style='margin-bottom: 2px;'>
                            <el-input type="textarea" v-model="addSnForm.eci"></el-input>
                        </el-form-item>
                        <span class='enbIdTip commonTitle12' v-show='eciTip'><%=rb.getString("ECIShuRuTiShi")%></span>
                        <el-form-item label="SN" prop="serialNumber" class='enbIdItem' style='margin-bottom: 2px; margin-top: 30px;'>
                            <el-input type="textarea" v-model="addSnForm.serialNumber"></el-input>
                        </el-form-item>
                        <span class='enbIdTip commonTitle12' v-show='snTip'><%=rb.getString("SNShuRuTiShi")%></span>
                       
                        <el-form-item label="MME IP" style='margin-top: 26px; margin-bottom: 6px;'>
							<el-input v-model='addSnForm.mme' style='width: 270px;'>
								 <i slot="append" class="el-icon el-icon-plus" @click="addMmeIp"></i>
							</el-input>	
							<div class='versionResultBox' v-show='addSnForm.mmeGroup.length > 0' style='width: 200px;'>
								<div class='suffixItem' v-for='(domain,index) in addSnForm.mmeGroup'>
									<div class='form-suffix' style='width: 170px; margin: 3px 10px 0;'>
										<span class='text'>{{domain}}</span>
										<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='removeMmeIp(domain)'></span>
									</div>
								</div>
							</div>
							<p class='ipErrorTip' style='margin-top: -5px;'>{{errorMmeIpMsg}}</p>
						</el-form-item>
						<el-form-item prop="mmeIp" style="margin-right: -1px;">
							<el-input v-model="addSnForm.mmeIp" v-show=false></el-input>
						</el-form-item>
                    </el-form>
   				</div>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="snAddSubmit" :disabled='snSaveBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="snAddCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
   </div>
	<el-dialog title="<%=rb.getString("XinXi")%>" width="400px" :visible="downloadVisible" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeDownloadDialog">
		<div>
			<p style="color:#797979;font-size:14px;"><%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%></p>
			<el-button style="margin-top:20px;margin-left:250px;" @click="downloadErrorFile" type="primary"><%=rb.getString("XiaZai")%></el-button>
		</div>
	</el-dialog>
    <%-- 表单-上传基站列表文件 --%>
	<form enctype="multipart/form-data" method="post" id="uploadForm_configPlan" style="display: none;">
		<input name="fileSize"  value="" hidden="true">
	    <input name="uploadFile" type="file" id="uploadFileConfigPlan">
	    <input name="operType" value="">
		<input name="policyId"  value="" hidden="true" v-if='curType == "modify"'>
	</form>
	<!-- batch import -->
    <el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition" :force-position="true" class='dialogIpsec'
	 :height="slideHeight" :modal='slideModal'  :width="slideWidth" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  
	 @cancel='cancelSlide' @ok='saveSlide'>
	 </el-slide>
	 <el-dialog title='SN List' width='800px' :visible.sync='snListShow' :append-to-body="true" :close-on-click-modal="false" @close='closeSnDialog'>
		<div slot="title" style="padding: 15px 0;display: flex;align-items: center;">
			<h5>SN List</h5> 
		</div>
		<el-ctable height="400" ref="snTableList" id="snTableList" style='border: 1px solid #E9EDF9;'
			:row-key="'serialNumber'" :time= '6' :url="snViewURL"
			@selection-change="selectChange"
			:query-params="snParams">
			<template slot="toolbar">
                <div style='display: flex;'>
                    <el-query type="normal" @query="queryDetect" placeholder="<%=rb.getString("Title_SheBeiBianMa")%>"></el-query> 
                    <div style='display: flex; margin-left: 20px; margin-top: 8px;'>
                    	<span><%=rb.getString("ZhiXianShiYIXiaFa")%></span> 
						<el-checkbox v-model="isExecute" label=" " true-label="1" false-label="0" :disabled="isReadOnly" style='margin: 3px 0 0 10px;'></el-checkbox>
                    </div>
                </div>
            </template>
			<el-table-column type="selection" :reserver-selection="true" v-if='curType != "view"'></el-table-column>
			<el-table-column label="<%=rb.getString("Title_SheBeiBianMa") %>" prop="serialNumber"></el-table-column>				
		</el-ctable>
		<div style="padding: 10px 0 0 10px;" v-if='curType != "view"'>
			<el-button type="primary" @click="delBatchSn" :disabled='snSelections.length == 0'><%=rb.getString("PiLiangShanChu") %> </el-button>
			<el-button @click="closeSnDialog"><%=rb.getString("QuXiao") %></el-button>
		</div>
	</el-dialog>
</div>
<script>
	var validateItem;
    var enbAddOrEditConfigVue = new Vue({
        el: '#addOrEditConfigPage',
        data() {
            var vm = this,
                nozhreg = '^(?:(?![\u4E00-\u9FA5]|[\uFE30-\uFFA0]).)*$',
                ipreg = '^(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|[1-9])\\.'
                        +'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
                        +'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)\\.'
                        +'(1\\d{2}|2[0-4]\\d|25[0-5]|[1-9]\\d|\\d)|%config$',
	            validatePolicyName = function(rule,value,callback){			  
					if(value === '' || value === null || value === undefined) {
						callback(new Error('<%=rb.getString("BiTian")%>'));
					}else{
						callback();
					}
				},
				//原始版本 software upgrade 开关打开校验必填项
				validateVersion = function(rule, value, callback) {
					var verType = vm.addOrEditForm.specifyVersionType;
					if(vm.addOrEditForm.upgradeEnable == '1'){
	                    if(verType == '1') {
	                    	callback();
	                    }else if(vm.resultOriginalVersionList.length == 0) {
	                    	callback('<%=rb.getString("QingXuanZeJiLu") %>');
	                    }else{
	                    	callback();
	                    }
                    }else{
                    	callback();
                    }					
                },
                //模板版本 software upgrade 开关打开校验必填项
                validateTarget = function(rule,value,callback) {
					if(vm.addOrEditForm.upgradeEnable == '1'){
	                	if( value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("BiTian")%>');
						}else {
							callback();
						}
					}else{
                    	callback();
                    }
				},
				validateFilesName = function(rule,value,callback) {
	            	value = vm.fileName;     		
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else {
						callback();
					}
				},
				// software upgrade  version
				validateAddVersion = function(rule, value, callback) {
					if(value) {
						callback();
                    }else {
                    	callback('<%=rb.getString("ZhiShaoXuanZeYiZhongBanBenFangShi")%>');
                    }
                },
				// reginon deploying
				validateTac = (rule,value,callback) => {
					var reg = /^(\d+)$/;
					var minVal = rule.min;
					var maxVal = rule.max;
					var tipStr = '<%=rb.getString("ZhengXing")%> <%=rb.getString("DouHao")%> '+minVal+' - '+maxVal;
					var msgStr = tipStr;
					var showFlag = false;
					
					if(value === "" || value === null || value === undefined){
						showFlag = true;
					}else{
						var showFlag = false;
						if (!(reg.test(value))) {
							showFlag = true;
						}else{
							
							if(minVal-value>0){
								showFlag = true;
							}
							if(maxVal-value<0){
								showFlag = true;
							}
						}
					}
					if(showFlag){
						callback(new Error(msgStr));
					}else{
						callback();
					}
			},
			validateIDRange = (rule,value,callback) => {
				var reg = /^[0-9]*$/;
				
				if (vm.idRangeStart=="" && vm.idRangeEnd ==""){
					callback();
				}else if (vm.idRangeStart && vm.idRangeEnd){
					if(reg.test(vm.idRangeStart) && reg.test(vm.idRangeEnd)){
						if (parseInt(vm.idRangeStart) > parseInt(vm.idRangeEnd)){
							callback('<%=rb.getString("IDFanWeiBuZhuiQue")%>');
						}else{
							callback();
						}
					}else {
						callback('<%=rb.getString("IDFanWeiBuZhuiQue")%>');
					}
				}else {
					callback('<%=rb.getString("IDFanWeiBuZhuiQue")%>');
				}

			},
			validateEnbId = function(rule,value,callback) { 				
				var curIdReg = /[^\d,]/g, curArr = value.split(',');
				
				if(value != null && value.length != 0 ){
					var enbIdFlag = curArr.every(function(item,index){
						return (!curIdReg.test(item) && (parseInt(item)>=0 && parseInt(item) <= 9999999)) 
					})
					if(enbIdFlag){
						vm.enbIdTip = true;
						callback();
					}else{
						vm.enbIdTip = false;
						callback(new Error('<%=rb.getString("eNBIDShuRuTiShi")%> <%=rb.getString("FanWei")%>:0-9999999'));
					}
				}else if (value == null || value.length == 0) {
					vm.enbIdTip = false;
					callback(new Error('<%=rb.getString("eNBIDShuRuTiShi")%>'))
				}else{
					vm.enbIdTip = true;
					callback();
				}
			},
			filePolygonValidate = (rule,value,callback) => {
				if(value) {
					if(value.length>100) {
						callback('<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>');
					}else if(!fileFormatMatch(value,"kml")){
						callback('<%=rb.getString("GeShiCuoWu")%>');
					}else {
						callback();
					}
				}else {
					callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
				}
			},
			//plan 
			validateFilePath = (rule,value,callback) => {
				if(value === "" || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXianXuanZeWenJian")%>'))
				}else if(!fileFormatMatch(value,"xlsx,xls,csv")){
					callback(new Error('<%=rb.getString("DaoRuWenJianGeShi")%>'))
				}else{
					callback();
				}
			},			
			validatePlmn = (rule,value,callback) => {
				var minVal = rule.min,
					maxVal = rule.max,
					mag = rule.mag,
					reg = /^-?\d+$/;
				
				if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
					if(value === "" || value === null || value === undefined){
						callback(new Error(mag));
					}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal)) && value.length<=6){
						callback();
					}else{
						callback(new Error(mag));
					}
				}else{
					if(value === "" || value === null || value === undefined){
						callback();//开关为 关时，不校验
					}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal)) && value.length<=6){
						callback();
					}else{
						callback(new Error(mag));
					}
				}
			},
			validateRootIndex = (rule,value,callback) => {
				if(vm.curProductName == 'NBIOT' || vm.curProductName == "DXDF"){
					callback();
				}else{
					var reg = /^(\d+,?)+$/,regRange = /^(\d+\.\.){0,1}(\d+)$/, minVal = rule.min, maxVal = rule.max, showFlag = false,
					tipStr = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
					if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
						if(value === "" || value === null || value === undefined){
							showFlag = true;
						}else{
							if (!(reg.test(value)) && !(regRange.test(value))) {
								showFlag = true;
							}else{
								//输入以逗号分隔的方式多输入
								var curValue = value.split(',');
								var newArrValue = curValue.filter(item => item);
								for(var i = 0; i < newArrValue.length; i++){
									if(newArrValue[i] > 837 || newArrValue[i] < 0){
										showFlag = true;
									}
								}
								if(value.indexOf("..")>-1){
									value= value.split("..");
									if(value[1]-value[0]<=0){
										showFlag = true;
									}
									if(minVal-value[0]>0){
										showFlag = true;
									}
									if(maxVal-value[1]<0){
										showFlag = true;
									}
								}else{
									if(minVal-value>0){
										showFlag = true;
									}
									if(maxVal-value<0){
										showFlag = true;
									}
								}
							};
						}
					}else{
						if(value === "" || value === null || value === undefined){
							showFlag = false;
						}else{
							if (!(reg.test(value)) && !(regRange.test(value))) {
								showFlag = true;
							}else{
								var curValue = value.split(',');
								for(var i = 0; i < curValue.length; i++){
									if(curValue[i] > 837 || curValue[i] < 0){
										showFlag = true;
									}
								}
								if(value.indexOf("..")>-1){
									value= value.split("..");
									if(value[1]-value[0]<=0){
										showFlag = true;
									}
									if(minVal-value[0]>0){
										showFlag = true;
									}
									if(maxVal-value[1]<0){
										showFlag = true;
									}
								}else{
									if(minVal-value>0){
										showFlag = true;
									}
									if(maxVal-value<0){
										showFlag = true;
									}
								}
							}
						}
					}
					
					if(showFlag){
						callback(new Error(tipStr));
					}else{
						callback();
					}
				}
			},
			validateRange = (rule,value,callback) => {
				if(vm.curProductName == 'DXDF'){
					var reg = /^-?\d+$/;
					if(rule.field == 'tac'){//频段
						var minVal = 0, maxVal = 65535;
						var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
							if(value === "" || value === null || value === undefined){
								callback(new Error(msgStr));
							}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
								callback();
							}else{
								callback(new Error(msgStr));
							}
						}else{
							if(value === "" || value === null || value === undefined){
								callback(); //开关为 关时，不校验
							}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
								callback();
							}else{
								callback(new Error(msgStr));
							}
						}
					}else if(rule.field == 'cell_identity'){
						//cell id
						var minVal = 0, maxVal = 268435455;
						var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
							if(value === "" || value === null || value === undefined){
								callback(new Error(msgStr));
							}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
								callback();
							}else{
								callback(new Error(msgStr));
							}
						}else{
							if(value === "" || value === null || value === undefined){
								callback(); //开关为 关时，不校验
							}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
								callback();
							}else{
								callback(new Error(msgStr));
							}
						}
					}else{
						//phycellid pci
						var minVal = 0, maxVal = 503;					
						var dxdfMsgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						var curIdReg = /[^\d,]/g, curArr = value.split(',');
						if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
							if(value != null && value.length != 0 ){
								var enbIdFlag = curArr.every(function(item,index){
									return (!curIdReg.test(item) && (parseInt(item)>= parseInt(minVal) && parseInt(item) <= parseInt(maxVal))) 
								})
								if(enbIdFlag){
									callback();
								}else{
									callback(new Error(dxdfMsgStr));
								}
							}else if (value == null || value.length == 0) {
								callback(new Error(dxdfMsgStr))
							}else{
								callback();
							}
						}else{
							if(value === "" || value === null || value === undefined){
								callback(); 
							}else{
								var enbIdFlag = curArr.every(function(item,index){
									return (!curIdReg.test(item) && (parseInt(item)>= parseInt(minVal) && parseInt(item) <= parseInt(maxVal))) 
								})
								if(enbIdFlag){
									callback();
								}else{
									callback(new Error(dxdfMsgStr));
								}
							}
						}
					}
				}else{
					var reg = /^(\d+\.\.){0,1}(\d+)$/, minVal = rule.min, maxVal = rule.max, showFlag = false, mag = rule.mag;
					if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
						if(value === "" || value === null || value === undefined){
							showFlag = true;
						}else{
							if (!(reg.test(value))) {
								showFlag = true;
							}else{
								if(value.indexOf("..")>-1){
									value= value.split("..");
									if(value[1]-value[0]<=0){
										showFlag = true;
									}
									if(minVal-value[0]>0){
										showFlag = true;
									}
									if(maxVal-value[1]<0){
										showFlag = true;
									}
								}else{
									if(minVal-value>0){
										showFlag = true;
									}
									if(maxVal-value<0){
										showFlag = true;
									}
								}
							}
						}
					}else{
						if(value === "" || value === null || value === undefined){
							showFlag = false;
						}else{
							if (!(reg.test(value))) {
								showFlag = true;
							}else{
								if(value.indexOf("..")>-1){
									value= value.split("..");
									if(value[1]-value[0]<=0){
										showFlag = true;
									}
									if(minVal-value[0]>0){
										showFlag = true;
									}
									if(maxVal-value[1]<0){
										showFlag = true;
									}
								}else{
									if(minVal-value>0){
										showFlag = true;
									}
									if(maxVal-value<0){
										showFlag = true;
									}
								}
							}
						}
					}
					
					if(showFlag){
						callback(new Error(mag));
					}else{
						callback();
					}
				}
			},
			validateIpsec = (rule,value,callback) => {
				if(vm.settingForm.ipsec_switch){
					if(vm.settingForm.ipsecList.length == 0){
						callback(new Error("IPSec Tunnel configure at least one"))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateIpsecEnable = (rule,value,callback) => {
				if(vm.settingForm.ipsec_switch){
					if(vm.settingForm.ipsec_enable == "" || vm.settingForm.ipsec_enable == null){
						callback(new Error('<%=rb.getString("QingXuanZe")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateRightikeport = (rule,value,callback) => {
				if(vm.settingForm.ipsec_switch){
					if(vm.settingForm.ipsec_rightikeport == "" || vm.settingForm.ipsec_rightikeport == null){
						callback(new Error('<%=rb.getString("QingXuanZe")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateInterface = (rule,value,callback) => {
				if(vm.settingForm.ipsec_switch){
					if(vm.settingForm.left_interface == "" || vm.settingForm.left_interface == null){
						callback(new Error('<%=rb.getString("QingXuanZe")%>'))
					}else{
						callback();
					}
				}else{
					callback();
				}
			},
			validateModuleType = (rule,value,callback) => {
				if(value.length) {
					callback();
				}else {
					vm.moduleErrorMessage = '';	
					callback('<%=rb.getString("BiTian")%>');
				}
			},
			validateBandwidth = (rule,value,callback) => {
				if(value) {
					callback();
				}else {
					callback('<%=rb.getString("BiTian")%>')
				}
			},
			validateInt = (rule,value,callback) => {
				if(vm.curProductName == 'DXDF'){
					var reg = /^-?\d+$/;
					if(rule.field == 'bands_support'){//频段
						var minVal = 1, maxVal = 85;
						var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
							if(value === "" || value === null || value === undefined){
								callback(new Error(msgStr));
							}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
								callback();
							}else{
								callback(new Error(msgStr));
							}
						}else{
							if(value === "" || value === null || value === undefined){
								callback(); //开关为 关时，不校验
							}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
								callback();
							}else{
								callback(new Error(msgStr));
							}
						}
					}else { //频点 frequency
						var minVal = 0, maxVal = 262143;					
						var dxdfMsgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						var curIdReg = /[^\d,]/g, curArr = value.split(',');
						if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
							if(value != null && value.length != 0 ){
								var enbIdFlag = curArr.every(function(item,index){
									return (!curIdReg.test(item) && (parseInt(item)>= parseInt(minVal) && parseInt(item) <= parseInt(maxVal))) 
								})
								if(enbIdFlag){
									callback();
								}else{
									callback(new Error(dxdfMsgStr));
								}
							}else if (value == null || value.length == 0) {
								callback(new Error(dxdfMsgStr))
							}else{
								callback();
							}
						}else{
							if(value === "" || value === null || value === undefined){
								callback(); 
							}else{
								var enbIdFlag = curArr.every(function(item,index){
									return (!curIdReg.test(item) && (parseInt(item)>= parseInt(minVal) && parseInt(item) <= parseInt(maxVal))) 
								})
								if(enbIdFlag){
									callback();
								}else{
									callback(new Error(dxdfMsgStr));
								}
							}
						}
					}
				}else{
					var minVal = rule.min,
					maxVal = rule.max,
					mag = rule.mag,
					reg = /^-?\d+$/;
					if(value != "" && value != null){
						var showFlag = value.indexOf("(");
						if(showFlag == -1){
							
						}else{
							value = value.substring(0,showFlag);
						}
					}
					
					if(vm.addOrEditForm.switchEnable == '1' && vm.addOrEditForm.selfConfigEnable == '1'){
						if(value === "" || value === null || value === undefined){
							callback(new Error(mag));
						}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
							callback();
						}else{
							callback(new Error(mag));
						}
					}else{
						if(value === "" || value === null || value === undefined){
							callback(); //开关为 关时，不校验
						}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
							callback();
						}else{
							callback(new Error(mag));
						}
					}
				}
			},
			validateIntEarfcn = (rule,value,callback) => {
				var minVal = rule.min,
					maxVal = rule.max;
				 	mag = rule.mag,
				 	reg = /^-?\d+$/;				 
				if(value != "" && value != null){
					var showFlag = value.indexOf("(");
					if(showFlag == -1){
						
					}else{
						value = value.substring(0,showFlag);
					}
				}
				
				if(value == ""){
					callback(new Error(mag));
				}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
					callback();
				}else{
					callback(new Error(mag));
				}
			},
			//参数配置-批量导入
			validateIntOne = (rule,value,callback) => {
				if(vm.platform != 'NBIOT' && rule.type == 'ulfrequency'){
					callback();
				}else if(vm.curProductName == 'DXDF'){
					var reg = /^-?\d+$/;
					if(rule.field == 'bands_support'){//频段
						var minVal = 1, maxVal = 85;
						var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						if(value === "" || value === null || value === undefined){
							callback(new Error(msgStr));
						}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
							callback();
						}else{
							callback(new Error(msgStr));
						}
					}else { //频点 frequency
						var minVal = 0, maxVal = 262143;					
						var dxdfMsgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						var curIdReg = /[^\d,]/g, curArr = value.split(',');
						if(value != null && value.length != 0 ){
							var enbIdFlag = curArr.every(function(item,index){
								return (!curIdReg.test(item) && (parseInt(item)>= parseInt(minVal) && parseInt(item) <= parseInt(maxVal))) 
							})
							if(enbIdFlag){
								callback();
							}else{
								callback(new Error(dxdfMsgStr));
							}
						}else if (value == null || value.length == 0) {
							callback(new Error(dxdfMsgStr))
						}else{
							callback();
						}
					}
				}else{
					if(value != "" && value != null){
						var showFlag = value.indexOf("(");
						if(showFlag == -1){
							
						}else{
							value = value.substring(0,showFlag);
						}
					}
					var minVal = rule.min;
					var maxVal = rule.max;
					var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
					var reg = /^-?\d+$/;
					if(value === "" || value === null || value === undefined){
						callback(new Error(msgStr));
					}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
						callback();
					}else{
						callback(new Error(msgStr));
					}
				}
			},
			validatePlmnOne = (rule,value,callback) => {
				var minVal = rule.min;
				var maxVal = rule.max;
				var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
				var reg = /^-?\d+$/;
				if(value === "" || value === null || value === undefined){
					callback(new Error(msgStr));
				}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal)) && value.length<=6){
					callback();
				}else{
					callback(new Error(msgStr));
				}
			},
			
			validateRangeIndex = (rule,value,callback) => {
				if(vm.platform == 'NBIOT' || vm.curProductName == "DXDF"){
					callback();
				}else{
					var reg = /^(\d+,?)+$/, regRange = /^(\d+\.\.){0,1}(\d+)$/, minVal = rule.min, maxVal = rule.max;
					var tipStr = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
					var msgStr = tipStr;
					var showFlag = false;
					if(value === "" || value === null || value === undefined){
						showFlag = true;
					}else{
						//var showFlag = false;
						if (!(reg.test(value)) && !(regRange.test(value))) {
							showFlag = true;
						}else{
							var curValue = value.split(',');
							var newArrValue = curValue.filter(item => item);
							for(var i = 0; i < newArrValue.length; i++){
								if(newArrValue[i] > 837 || newArrValue[i] < 0){
									showFlag = true;
								}
							}
							if(value.indexOf("..")>-1){
								value= value.split("..");
								if(value[1]-value[0]<=0){
									showFlag = true;
								}
								if(minVal-value[0]>0){
									showFlag = true;
								}
								if(maxVal-value[1]<0){
									showFlag = true;
								}
							}else{
								if(minVal-value>0){
									showFlag = true;
								}
								if(maxVal-value<0){
									showFlag = true;
								}
							}
						}
					}
					if(showFlag){
						callback(new Error(msgStr));
					}else{
						callback();
					}
				}
			},
			validateRangeOne = (rule,value,callback) => {
				if(vm.curProductName == 'DXDF'){
					var reg = /^-?\d+$/;
					if(rule.field == 'tac'){//频段
						var minVal = 0, maxVal = 65535;
						var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						if(value === "" || value === null || value === undefined){
							callback(new Error(msgStr));
						}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
							callback();
						}else{
							callback(new Error(msgStr));
						}
					}else if(rule.field == 'cell_identity'){
						//cell id
						var minVal = 0, maxVal = 268435455;
						var msgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						if(value === "" || value === null || value === undefined){
							callback(new Error(msgStr));
						}else if(reg.test(value) && (value >= parseInt(minVal)) && (value <= parseInt(maxVal))){
							callback();
						}else{
							callback(new Error(msgStr));
						}
					}else{
						//phycellid pci
						var minVal = 0, maxVal = 503;					
						var dxdfMsgStr = '<%=rb.getString("ZhengXing")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
						var curIdReg = /[^\d,]/g;
						if(value != null && value.length != 0 ){
							var curArr = value.split(',');
							var enbIdFlag = curArr.every(function(item,index){
								return (!curIdReg.test(item) && (parseInt(item)>= parseInt(minVal) && parseInt(item) <= parseInt(maxVal))) 
							})
							if(enbIdFlag){
								callback();
							}else{
								callback(new Error(dxdfMsgStr));
							}
						}else if (value == null || value.length == 0) {
							callback(new Error(dxdfMsgStr))
						}else{
							callback();
						}
					}
				}else{
					var reg = /^(\d+\.\.){0,1}(\d+)$/;
					var minVal = rule.min;
					var maxVal = rule.max;
					var tipStr = '<%=rb.getString("ZhengXing")%>/<%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%><%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
					if(rule.type == 'auto'){
						var msgStr = 'AUTO|'+tipStr;
					}else{
						var msgStr = tipStr;
					}
					var showFlag = false;
					if(value === "" || value === null || value === undefined){
						showFlag = true;
					}else{
						var showFlag = false;
						if (!(reg.test(value) || value == 'AUTO')) {
							showFlag = true;
						}else{
							if(value.indexOf("..")>-1){
								value= value.split("..");
								if(value[1]-value[0]<=0){
									showFlag = true;
								}
								if(minVal-value[0]>0){
									showFlag = true;
								}
								if(maxVal-value[1]<0){
									showFlag = true;
								}
							}else{
								if(minVal-value>0){
									showFlag = true;
								}
								if(maxVal-value<0){
									showFlag = true;
								}
							}
						}
					}
					if(showFlag){
						callback(new Error(msgStr));
					}else{
						callback();
					}
				}
			},
			validateGroupName = function(rule,value,callback){			  
				if(value === '' || value === null || value === undefined) {
					callback(new Error('<%=rb.getString("BiTian")%>'));
				}else{
					callback();
				}
			},
			//regRange = /^(\d+\.\.){0,1}(\d+)$/;regRange = /^\d+(?:..\d)*$/;
			validateEci = function(rule,value,callback) { 				
				var reg = /^(\d+)$/, regRange = /^(\d+\.\.){0,1}(\d+)$/;
				var minVal = rule.min;
				var maxVal = rule.max;
				var tipStr = '<%=rb.getString("ZhengXing")%> / <%=rb.getString("FanWei")%><%=rb.getString("DouHao")%><%=rb.getString("ZuiXiaoZhi")%><%=rb.getString("MaoHao")%>'+minVal+'<%=rb.getString("DouHao")%> <%=rb.getString("ZuiDaZhi")%><%=rb.getString("MaoHao")%>'+maxVal;
				var msgStr = tipStr;
				var showFlag = false;
				if(value === "" || value === null || value === undefined){
					showFlag = true;
				}else{
					var curValue = value.split(',');
					curValue.map(function(item){
						if((regRange.test(item))){
							var value = item.split("..");
							if(item.indexOf("..") >-1){
								if(value[1]-value[0]<=0){
									showFlag = true;
								}
								if(minVal-value[0]>0){
									showFlag = true;
								}
								if(maxVal-value[1]<0){
									showFlag = true;
								}
							}else{
								if(value[0] > maxVal || value[0] < minVal || value[0] === '' || value[0] ===null || value[0] ===undefined){
	                        		showFlag = true;
	                        	}
							}
                        }else{
                        	if(!(reg.test(item))){
	                        	showFlag = true;
	                        }else{
	                        	if(item > maxVal || item < minVal){
	                        		showFlag = true;
	                        	}
	                        }
                        }
                    });
				}
				if(showFlag){
					vm.eciTip = false;
					callback(new Error(msgStr));
				}else{
					vm.eciTip = true;
					callback();
				}
			},
			validateSn = function(rule,value,callback) { 
				var serialNumber = value, list = [], temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/;
					list = serialNumber.split(',');
			    if(serialNumber != null && serialNumber.length != 0){
					var nameFlag = list.every(function(item,index){
						return temp.test(item)
					})
					if(nameFlag){
						vm.snTip = true;
						callback()
					}else{
						vm.snTip = false;
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
				}else if (serialNumber == null || serialNumber.length == 0) {
					vm.snTip = false;
					callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
				}else{
					vm.snTip = true;
					callback();
				}
			},
			validateMmeIp = function(rule,value,callback) {
				if(vm.addSnForm.mmeGroup.length == 0 ){
					vm.errorMmeIpMsg = '';
					callback(new Error('<%=rb.getString("IPDiZhi")%>'))
				}else{
					callback();
				}
			},
			validateIPAddress = (rule,value,callback)=>{
				var regIPAddress = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-4]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/,
					isWanEnabled = vm.wanConfigForm && vm.wanConfigForm.wanSendEnable === '1';
				if(!value){
					if(isWanEnabled){
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'));
					}else{
						callback();
					}
				}else{
					if(!regIPAddress.test(value)){
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'));
					}else{
						callback();
					}
				}
            },
			validateNetmask = (rule,value,callback)=>{
				var isWanEnabled = vm.wanConfigForm && vm.wanConfigForm.wanSendEnable === '1';
				if(!value){
					if(isWanEnabled){
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'));
					}else{
						callback();
					}
				}else{
					if(!vm.isMask(value)){
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'));
					}else{
						callback();
					}
				}
            },
			validateGateway = (rule,value,callback) => {
                if(!value || vm.isValidIP(value)){
					callback();
				}else{
					callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
				}
            },
			validateVlanId = (rule,value,callback)=>{
                var min = rule.min, max = rule.max, reg = /^(\d+\.\.){0,1}(\d+)$/;
                if(!value){
					callback();
				}else if(!reg.test(value) || value < min || value > max){
					callback(new Error('<%=rb.getString("FanWei")%>：1-4094,<%=rb.getString("ZhengXing")%>'))
				}else{
					callback();
				}
            },
			// WAN Binding 专用验证函数
			validateWanBinding = (rule,value,callback) => {
				var isWanEnabled = vm.wanConfigForm && vm.wanConfigForm.wanSendEnable === '1';
				if(!value){
					if(isWanEnabled){
						callback(new Error('<%=rb.getString("BiTian")%>'));
					}else{
						callback();
					}
				}else{
					callback();
				}
			};
            return {
				// -------------------------------advance settings
				msg: {
					smhd: '<%=rb.getString("ShuZiJiaSMHD")%> (d:[0-365]、h:[0-8760]、m:[0-525600]、s:[0-31536000])',
					noZh: '<%=rb.getString("FeiZhongWenZiFu")%> <%=rb.getString("MingChengChangDu")%>'
				},
				advanceTreeShow: false,
				extend: {},
				paramKeys: [],
				advanceForm: {
					advanceEnable: '0',
					ipsec_enable: '0',
					ipsecList: [],
					mtu: '',
					totalTxPower: '',
					powerRamping: '',
					preambleInitTargetPower: '',
					poNominalPusch: '',
					poNominalPucch: '',
					alpha: '',
					targetUlSinr: '',
					powerPA: '',
					powerPB: '',
					dnsHostName: '',
					dnsTimeZone: '',
					dnsAddress1: '',
					dnsAddress2: '',
					dnsAddress3: '',
					ntpEnable: '0',
					ntpPort1: '',
					ntpServer1: '',
					ntpPort2: '',
					ntpServer2: '',
					ntpPort3: '',
					ntpServer3: ''
				},
				advanceTree: [
					{
						id: 'wan',
						label: 'WAN',
						children: [ { id: 'wanConfig', label: 'WAN Config' } ]
					},
					{
						id: 'ipsec',
						label: 'IPSec',
						children: [ { id: 'ipsecSettings', label: 'IPSec Settings' } ]
					},
					{	
						id: 'power',
						label: 'Power Control',
						children: [ { id: 'powerControl', label: 'Power Control Parameters' } ]
					},
					{
						id: 'system',
						label: 'System',
						children: [ { id: 'dns', label: 'DNS' }, { id: 'ntp', label: 'NTP' } ]
					}
				],
				advanceTreeSelected: [],
				defaultRreeChecked: [],
				addOrEditForm: {
            		selfStartEnable: '1',//总开关
            		policyName: '',
            		productType: '',
            		executeType: '0',
            		//开关默认打开
            		upgradeEnable: '0',//software upgrade enable
            		specifyVersionType: '0', //any checkbox
            		targetVersion: '',
            		originalVersion: '',
            		preserveSetting: '1',
            		licenseEnable: '0', // license enable
            		// parameters config
            		selfConfigEnable: '0', //parameters config enable         		
            		switchEnable: '0', //parameters data pool enable
            		planConfigEnable: '0', //batch import enable
            		selfAutoMapEnable: '0', // region deploying  enable
            		autoConfigMode: '0', //coordinate-TAC, eNB ID-TAC radio
            		defaultTac: '0', //eNB ID-TAC checkbox
            		//参数配置涉及参数
            		bands_support:"",
					band_width:"n100",
					frequency:"",
					ul_frequency:"",
					subframe_assignment:"",
					special_subframe_patterns:"",
					plmn_id:"",
					tac:"",
					cell_identity:"",
					phycellid:"",
					root_sequence_index:"",
					mme:"",
					mmeGroup:[],
					oldMmeArr: [],
					mmeStr:"",
					halob_enable:"",
					carrier_mode:"",
					moduleTypeList: [],
					module_enable: '',
					custParam: [], 
					txPower: []
				},
				addOrEditRule: {
					policyName: [{validator: validatePolicyName}],
					productType: [{validator: validatePolicyName}],
					originalVersion: [{validator: validateVersion}],
					targetVersion: [{validator: validateTarget}],
					bands_support:[{validator: validateInt, min:1, max:62, mag:'<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 1~62', trigger:'blur'}],
					frequency:[{validator: validateInt, min:0, max:65535, mag:'<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 1~65535', trigger:'blur'}],
					ul_frequency:[{validator: validateInt, min:0, max:65535, mag:'<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 1~65535', trigger:'blur'}],
					plmn_id:[{validator: validatePlmn,min:00000,max:999999, mag:'<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 00000~999999', trigger:'blur'}],
					tac:[{validator: validateRange,min:0,max:65535, mag:'<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 0~65535', trigger:'blur'}],
					cell_identity:[{validator: validateRange,min:0,max:268435455, mag:'<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 0~268435455', trigger:'blur'}],
					phycellid:[{validator: validateRange,min:0,max:503, mag:'<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>: 0~503', trigger:'blur'}],
					root_sequence_index:[{validator: validateRootIndex,min:0,max:837,trigger:'blur', type:'rootIndex'}]
				},
				productList: [],
				errorMsgOne:'',
				//----------------------------------software upgrade
				//version
            	tarVersionList: [],
                addVersionShow: false,
                addVersionBtnShow: true,
                resultOriginalVersionShow: false,
            	softwareAddVersionForm: {
            		versionStr: '',
					versionList: [],
					itemTest:'',           	
					originalVersion: '',
            	},
            	softwareAddVersionRules: {},
            	versionErrorMessage: '',
            	versionMessage: '',
            	resultOriginalVersionList: [],
            	softwareOriginalVersionList: [],
				curSelectVersionData: [],            
				isReadOnly: false,
                messages: {
                    placeholder: '<%=rb.getString("ChuShiBanBen") %>'
                },
                longLength: 200,
                searchText: '',
                qataLimit:true,                
                enable: '0',
                enbOriginalQuery: {
                	searchText: '',
					timeZone: timeZone
                },
                searchValue: "",
                //---------------------------------------------license
                licenseImportShow: false,
                licenseUrl: '${ctx}/SON/SelfConfiguration/queryPNPLicensePageList.action',
                height:"100%",
                enbLicenseQuery: {
					searchText: '',
					productType: '',
					timeZone: timeZone
				},
				importModeSelection:'moreFile',
	            fileName:'',
	            licenseForm: {
	                uploadFileUrl: '',
	           	},
	            moreFileParams:{},
				newFileList:[],
	            moreFileList:[],
	            fileData:[],
				importLicenseRules: {	           		
					fileName:[{validator: validateFilesName}] 
	            },
	            uploadBtnDisabled: false,
				licenseLoading: false,
	            //---------------------------------------parameter config
	            //params pool
	            moduleTitle: '<%=rb.getString("TianJia")%>',
	            moduleFlag: 'add',
	            modelTypeAddShow: false,
            	moduleVal: '',
            	moduleForm: {
					module_type: [],
					band_width: '',
					frequency: ''
				},
				moduleRules: {
					band_width: [{validator: validateBandwidth}],
					frequency:[{validator: validateIntEarfcn,min: 0, max: 65535,mag:'<%=rb.getString("ZhengXing")%>,<%=rb.getString("FanWei")%>：0~65535', trigger:'blur'}] 
				},
				menus_module: [],
				moduleErrorMessage: '',
	            functionModulesSelect: '0',
                parameterConfigActive: 'paramOne',
                activeNameOne:['basic','coreNetwork','customized'],
				limitForm:{
					band_width:true,
					frequency:true,
					ul_frequency:true,
					subframe_assignment:true,
					special_subframe_patterns:true,
					root_sequence_index:true,
					halob_enable:true,
					carrier_mode:true,
				},
				showMMEMain: false,
				qataLimitMain: true,
                //batch import
                specifyImportShow: false,
                importForm: {
					importType: "0",
					filePath: ""
				},
				importRules: {
					filePath: [{validator: validateFilePath}] 
				},
				exportTypeOptions: [{id:"intel",text:"R Series"},{id:"qa",text:"Q Series"},{id:'NBIOT',text:"NB Series"}],
				exportTypeFlag: false,
				importLoading: false,
				mapLoading: false,
				importSpecialLoading: false,
				mapSubmitDisabled: false,
				downloadVisible: false,
				//修改
                specifyModifyShow: false,
                curSpecifyModifyRow: [],
                activeNames:['basic', 'cell'],
                basicForm:{
					bands_support:"",
					band_width:"",
					frequency:"",
					ul_frequency:"",
					subframe_assignment:"",
					special_subframe_patterns:"",
					plmn_id:"",
					tac:"",
					cell_identity:"",
					phycellid:"",
					root_sequence_index:"",
					mme:"",
					mmeGroup:[],
					mmeStr:"",
					halob_enable:"",
					carrier_mode:"",
					timeZoneUtc:'',
					txPower:'',
					dxdfTxPower: [],
					//cell2
					cellPlmnId: '',
					cellBandwidth: '', //带宽
					cellFrequency: '',//频点
					cellEci: '',// eci
					cellPhycellid: '',//pci
					cellRootIndex: '',
					cellMme: '',
					cellMmeGroup: [],
					cellMmeStr: '',
				},
				basicRules:{
					bands_support:[{validator: validateIntOne,min:1,max:62,trigger:'blur'}],
					frequency:[{validator: validateIntOne,min:0,max:65535,trigger:'blur'}],
					plmn_id:[{validator: validatePlmnOne,min:0,max:999999,trigger:'blur'}],
					tac:[{validator: validateRangeOne,min:0,max:65535,trigger:'blur',type:'auto'}],
					cell_identity:[{validator: validateRangeOne,min:0,max:268435455,trigger:'blur',type:'auto'}],
					phycellid:[{validator: validateRangeOne,min:0,max:503,trigger:'blur'}],
					root_sequence_index:[{validator: validateRangeIndex,min:0,max:837,trigger:'blur',type:'rootIndex'}],
					//CELL2
					cellPlmnId:[{validator: validatePlmnOne,min:0,max:999999,trigger:'blur'}],
					cellFrequency:[{validator: validateIntOne,min:0,max:65535,trigger:'blur'}],
					cellEci:[{validator: validateRangeOne,min:0,max:268435455,trigger:'blur',type:'auto'}],
					cellPhycellid:[{validator: validateRangeOne,min:0,max:503,trigger:'blur'}],
					cellRootIndex:[{validator: validateRangeIndex,min:0,max:837,trigger:'blur',type:'rootIndex'}],	
				},
				errorMsg:"",
				cellErrorMsg: '',
				showCell2: false,
				modifyBasicTitle: '',
				//settings
				menus_ipsec:[],
				rowDataIpsec:[],
				settingForm:{
					ipsec_switch:false,
					ipsec_enable:"1",
					ipsec_rightikeport:"",
					left_interface:"",
					ipsecList:[],
					defaultIpsecList:[],
					ipsecLength:""
				},
				settingRules:{
					ipsecLength:[{validator: validateIpsec}],
					ipsec_enable:[{validator: validateIpsecEnable}],
					ipsec_rightikeport:[{validator: validateRightikeport}],
					left_interface:[{validator: validateInterface}],
				},
				wanConfigForm: {
					wanSendEnable: '0',
					wanBindingList: [
						{wanIndex: 1, ipAddress: '', netmask: '', gateway: '', vlanId: '', wanBinding: ''},
						{wanIndex: 2, ipAddress: '', netmask: '', gateway: '', vlanId: '', wanBinding: ''},
						{wanIndex: 3, ipAddress: '', netmask: '', gateway: '', vlanId: '', wanBinding: ''},
						{wanIndex: 4, ipAddress: '', netmask: '', gateway: '', vlanId: '', wanBinding: ''}
					],
					originalWanBindingList: []
				},
				wanConfigRules: (function() {
					// 使用立即执行函数生成规则，避免引用未定义的验证函数
					var rules = {};
					for(var i = 0; i < 4; i++) {
						var prefix = 'wanBindingList.' + i + '.';
						rules[prefix + 'ipAddress'] = [{required: true, validator: validateIPAddress}];
						rules[prefix + 'netmask'] = [{required: true, validator: validateNetmask}];
						rules[prefix + 'wanBinding'] = [{required: true, validator: validateWanBinding}];
						rules[prefix + 'gateway'] = [{validator: validateGateway}];
						rules[prefix + 'vlanId'] = [{validator: validateVlanId, min:1, max:4094}];
					}
					return rules;
				})(),
				wanNameList: ['WAN(OAM-TR069)', 'WAN(S1-C)', 'WAN(S1-U)', 'WAN(X2AP)'],
				wanBindingLabelList: ['TR069 Binding', 'S1-C Binding', 'S1-U Binding', 'X2-ap Binding'],
				//settings table ：add or edit
				ipsecDlShow: false,
				paramList: [],
				ipsecForm: {},
				ipsecTunnelTitle: '<%=rb.getString("TianJia")%> Ipsec Tunnel',
				viewDisabled: false,
				ipsecRules: {},
				operTypeIpsec: "",
				rowDataPlan: [],
                //修改，详情 共用
                sasDisabled: false,
                showSetting:false,
				showBasic: true,
				showMME:false,
				showMode:false,
				showItem:false,
				showPtp:false,
				showAddress:false,
				shownbi: true,
				mapFlag: false,
				timeZoneList:[],
				powerList:[],
				dxdfPowerList: [],
				frequencyLabel:'<%=rb.getString("ENBPinDian")%>',
              	//详情
                specifyInfoShow: false,
                ipsecInfoSasDisabled: false,
                ipsecInfoShowSetting: false,
                ipsecInfoShowBasic: true,
                ipsecInfoShowMME: false,
                ipsecInfoShowMode: false,
                ipsecInfoShowItem: false,
                ipsecInfoShowPtp: false,
                ipsecInfoShowAddress: false,
                ipsecInfoShownbi: true,
                ipsecInfoMapFlag: false,
                infoActiveName: ['basic', 'cell'],
                specifySN: '',
                cellName: '',
                //basic config
                bands_support: '',
                band_width: '',
                frequency: '',
                subframe_assignment: '',
                special_subframe_patterns: '',
                plmn_id: '',
                tac: '',
                cell_identity: '',
                phycellid: '',
                root_sequence_index: '',
                txPower: '',
                halob_enable: '',
                mmeStr: '',
                carrier_mode: '',
                timeZoneUtc: '',
				//cell2
				cellPlmnId: '',
				cellBandwidth: '', //带宽
				cellFrequency: '',//频点
				cellEci: '',// eci
				cellPhycellid: '',//pci
				cellRootIndex: '',
				cellInfoMme: '',
				showInfoCell2: false,
                //settings
                ipsec_switch:false,
				ipsec_enable:"1",
				ipsec_rightikeport:"",
				left_interface:"",
				ipsecList:[],
				slideUrl:"",
				slideTitle:"",
				slideFooter:"",
				slideHeader:"",
				slidePosition:"",
				slideHeight:"",
				slideWidth:"",
				slideModal:"",               
				infoWanSendEnable: '0',
				infoWanBindingList: [],              
                urlConfigPlan:"",
				params_plan:{
					timeZone: timeZone,
					searchText: "",
					policyId: ''
				},
				params_plan_form:{searchText:""},
                //Region deploying 
                coordinateTacImportShow: false,
                coordinateTacModifyShow: false,
                enbIdTacModifyShow: false,
                mapForm:{
					file_polygon:''
				},
				mapRule:{
					file_polygon:[{validator: filePolygonValidate}]
				},
				showMapImport:false,
				file_polygon:'',
                AddModifyTacTitle: '<%=rb.getString("TianJia")%>',
                enbIdUrl: '${ctx}/SON/SelfConfiguration/querySelfConfigTacPageList.action',
                urlMap:"${ctx}/SON/SelfConfiguration/querySelfCoordinateTacPageList.action",
                modifyMapForm:{
					groupName:'',
					tac:'',
					idRange:''
				},
				idRangeStart:'',
				idRangeEnd:'',
				editMapRule:{
					tac:[{validator: validateTac,min:0,max:65535,trigger:'blur'}],
					idRange:[{validator: validateIDRange,trigger:'blur'}]
				},				
                addTacForm: {
					tac: '',
					eciRange:"",
				},
				addTacRule: {
					tac: [{validator: validateTac, min: 0, max: 65535, trigger: 'blur'}],
					eciRange: [{validator: validateEnbId, min:0,max:1048575,mag:'<%=rb.getString("FanWei")%>：0~1048575'}],
				},
				enbIdTacRowData: [],
				menus_enbIdtac: [],
				enbIdTip: true,
				enbIdTacDisabled: false,
				operTacType: 'add',
                policyTitle: 'Add Policy',
                curType: '',
                curPolicyId: '',
                platform: '',
                curProductName: '',
                saveBtnDisabled: false,
                //location-sn
                snUrl: '${ctx}/SON/SelfConfiguration/querySelfAreaTacPageList.action',
                snAddShow: false,
                addSnForm: {
                	groupName:'',
					tac: '',
					eci: '',
					serialNumber: '',
					mme:"",
					mmeGroup:[],
					mmeStr:""
				},
				addSnRule: {
					tac: [{validator: validateTac, min: 0, max: 65535, trigger: 'blur'}],
					eci: [{validator: validateEci ,min:0, max:268435455,trigger: 'blur'}],
					serialNumber: [{validator: validateSn}],
					groupName: [{validator: validateGroupName}],
				},
				errorMmeIpMsg: '',
				snListShow: false,
				snViewURL: '${ctx}/SON/SelfConfiguration/querySelfSNTacPageList.action',
				snParams: {
					policyId: '',
					searchText: '',
					groupName: '',
					isExecute: ''
				},
				snSelections: [],
				snQueryParam: {
					searchText: "",
					policyId: ''
				},
				eciTip: true,
				snTip: true,
				snSaveBtnDisabled: false,
				operSnType: 'add',
				AddModifySnTitle: '<%=rb.getString("TianJia")%>',
				snNameDisabled: false,
				snRowData: [],
				isExecute: '0',
				eciLable: '',
				plmnGroup:[],
				plmnVal:'',
				plmnCls:'',
            }
        },
		
        computed: {
			groupShow() {
				var vm = this,
					selected = vm.productList.filter(function(item){
						return item.value == vm.addOrEditForm.productType;
					})[0],
					type = selected?selected.name:'';
				return {
					wan: ['RTS','RTD'].includes(type),
					ipsec: ['RTS','RTD','QRTB','BAIBLQ','MLQ','BLX','QAFA','QATA','QAFB','PM-B4860','CR-B4860/BU','CR-B4860/EU','CR-B4860/RU'].includes(type),
					power: ['RTS','RTD','QRTB','BAIBLQ','MLQ','BLX','QAFA','QATA','QAFB'].includes(type),
					dns: ['QAFA','QATA','QAFB'].includes(type),
					ntp: ['RTS','RTD','QRTB','BAIBLQ','MLQ','BLX','QAFA','QATA','QAFB','PM-B4860','CR-B4860/BU','CR-B4860/EU','CR-B4860/RU'].includes(type)
				}
			},
			isBLX() {
				var vm = this,
					selected = vm.productList.filter(function(item){
						return item.value == vm.addOrEditForm.productType;
					})[0],
					type = selected?selected.name:'';
				return ['BLX'].includes(type);
			}
        },
        watch: {
            'addOrEditForm.productType': function(val) {
                var vm = this, params = { product_value: val };
                if(val == 'FAP/\\w+(BS41)\\w+/SC'){
                	vm.qataLimit = false;
                }
                // 获取目标版本信息
                axios.post("${ctx}/plugandplay/policy/getTargetVersion.action", stringify(params)).then(function(response){
					var data = response.data;
					vm.tarVersionList = data || [];
				});
                //产品类型改变，license 查询时product 参数改变；
                vm.enbLicenseQuery.productType = val;
                //Nova430i 目前没有 'CR-B4860/EU','CR-B4860/RU' 去除
                var productNameList = ['DXDF','RTS','RTD','QRTB','BAIBLQ','BLX','MLQ','QATA','QAFB','QAFA','NBIOT','PM-B4860','CR-B4860/BU'];
                //判断产品类型的name 是否和productNameList 数据相同；
                for(var i = 0; i<vm.productList.length; i++){
                	for(var j = 0; j<productNameList.length; j++){
                		if(vm.productList[i].value == val){
                			if(vm.productList[i].name == productNameList[j]){
                    			if(vm.productList[i].name == 'PM-B4860' || vm.productList[i].name == 'CR-B4860/BU'){
                    				vm.curProductName = 'CR-B4860';
									vm.modifyBasicTitle = '<%=rb.getString("XiaoQuCanShu")%> 1';
                    			}else{
                    				vm.curProductName = vm.productList[i].name;
									vm.modifyBasicTitle = '<%=rb.getString("JiChuPeiZhi")%>';
                    			}
                    		}
                		}               		
                	}
                }	
				if(vm.curProductName == 'DXDF'){
					if(vm.curType == 'add'){
						vm.addOrEditForm.band_width = '6';
					}
					vm.eciLable = 'CELL ID';
				}else {
					if(vm.curType == 'add'){
						vm.addOrEditForm.band_width = 'n100';
					}
					vm.eciLable = 'ECI (ECI=eNB_ID*256+Cell_ID)';
				}		
				//参数配置：参数数据池模块 内容回显, 未选择产品类型时，按照 ''RTS'内容回显及展示 ; 经确认新建时 不需回显
				vm.commonParamConfigContent();
            },
            //升级开关关闭，不校验必填项，输入框可置灰
            'addOrEditForm.upgradeEnable': function(val) {
                var vm = this;
                if(val == '1'){
                	vm.$refs.addOrEditForm.validateField('originalVersion');
                	vm.$refs.addOrEditForm.validateField('targetVersion');
                }else{
                	vm.$refs.addOrEditForm.clearValidate('originalVersion');
                	vm.$refs.addOrEditForm.clearValidate('targetVersion');
                	vm.addVersionShow = false;
                }                    
            },
            //参数数据池开关
            "addOrEditForm.halob_enable": function(val) {
				var vm = this;		
				vm.showMMEMain = val == "1" ? false : true;
            },
    		curSelectVersionData(row){
    			if(row.length != 0){
    				this.versionMessage = '';
    			}    			
    		},
    		//初始版本： 0-非all, 1- all 初始版本不可添加，
    		'addOrEditForm.specifyVersionType': function(val) {
                 if(val == '0'){
                	this.$refs.addOrEditForm.validateField('originalVersion');
                 }else{
                	 this.addVersionShow = false;
                	 this.resultOriginalVersionList = []; //置空已配置的初始版本
                	 this.$refs.addOrEditForm.clearValidate('originalVersion');
                 }
            },
            "idRangeStart":function(val){
				this.modifyMapForm.idRange = this.idRangeStart + "-" + this.idRangeEnd
			},
			"idRangeEnd":function(val){
				this.modifyMapForm.idRange = this.idRangeStart + "-" + this.idRangeEnd
			},
			"basicForm.mmeGroup":function(){
				this.basicForm.mmeStr = this.basicForm.mmeGroup.toString();
			},	
			"basicForm.cellMmeGroup":function(){
				this.basicForm.cellMmeStr = this.basicForm.cellMmeGroup.toString();
			},			
			isExecute:function(val){
				var vm = this;
				vm.snParams.isExecute = val;
			},
			"addSnForm.mmeGroup":function(){
				this.addSnForm.mmeIp = this.addSnForm.mmeGroup.toString();
			},
			'wanConfigForm.wanSendEnable': function(newVal) {
				if(newVal === '0') {
					this.$refs.wanConfigForm.clearValidate();
				}
			}
        },
        methods: {
            enbInit(type, id) {
                var vm = this,
					powerLists = ['-60dBm','-50dBm','-40dBm','-30dBm','-20dBm','-10dBm','0dBm','1dBm','2dBm','3dBm','4dBm','5dBm','6dBm','7dBm','8dBm','9dBm','10dBm','11dBm','12dBm','13dBm','14dBm','15dBm','16dBm','17dBm','18dBm','19dBm','20dBm','21dBm','22dBm','23dBm','24dBm','25dBm','26dBm','27dBm','28dBm','29dBm','30dBm','31dBm','32dBm','33dBm','34dBm','35dBm','36dBm','37dBm','38dBm','39dBm','40dBm','41dBm','42dBm','43dBm','44dBm','45dBm','46dBm','47dBm','48dBm','49dBm','50dBm'],
                	powerValLists = [-60,-50,-40,-30,-20,-10,0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50];
				vm.dxdfPowerList = powerLists.map((item,index)=>{
					return {label:item,value:powerValLists[index]}
				})
                vm.curPolicyId = id;
				vm.isReadOnly = type == 'readonly';
                if(type == 'add'){
                	vm.policyTitle = '<%=rb.getString("XinZengCeLue")%>';
                	vm.curType = 'add'
                }else if(type == 'modify'){
                	vm.policyTitle = '<%=rb.getString("XiuGaiCeLue")%>';
                	vm.curType = 'modify';
                	vm.params_plan.policyId = id;
                }else{
                	vm.policyTitle = '<%=rb.getString("ChaKanCeLue")%>';
                	vm.curType = 'view';
                }
                //详情，修改时需回显数据
                //axios.post("${ctx}/task/upgrade/getProductType.action",stringify({
                axios.post("${ctx}/SON/SelfConfiguration/getProductTypeSelect.action",stringify({
                	type: 'all'
                })).then(function(res){
                	var data = res.data;
                	//产品类型是否存在没有的情况
                	if(data.length > 0){
                		vm.productList = data.filter(function(item,index){
    						return item.value != 'CR-B4860/EU' && item.value != 'CR-B4860/RU'
    					})
    					if(type == 'add'){
    						vm.addOrEditForm.productType = vm.productList[0].value;
    						vm.curProductName = vm.productList[0].name;
    						// init advance params
    						vm.initParamKeys(); 
    					}
    					if(vm.isReadOnly || type == 'modify') {
							vm.snQueryParam.policyId = id; 
    						vm.getInfoById(id);
    					}
                	}
				});   
            },
			getDefaultWanBindingList(){
				return [
					{wanIndex: 1, ipAddress: '', netmask: '', gateway: '', vlanId: '', wanBinding: ''},
					{wanIndex: 2, ipAddress: '', netmask: '', gateway: '', vlanId: '', wanBinding: ''},
					{wanIndex: 3, ipAddress: '', netmask: '', gateway: '', vlanId: '', wanBinding: ''},
					{wanIndex: 4, ipAddress: '', netmask: '', gateway: '', vlanId: '', wanBinding: ''}
				];
			},
			// 合并接口数据和默认列表，确保所有 wanIndex 都存在且顺序正确
			mergeWanBindingList(apiData){
				// 获取默认的完整列表（wanIndex 1-4）
				var defaultList = this.getDefaultWanBindingList();
				
				// 如果接口没有返回数据或为空数组，直接返回默认列表
				if(!apiData || apiData.length === 0){
					return defaultList;
				}
				
				// 创建一个 Map 来存储接口返回的数据，key 为 wanIndex
				var apiDataMap = {};
				apiData.forEach(function(item){
					if(item.wanIndex){
						apiDataMap[item.wanIndex] = item;
					}
				});
				
				// 遍历默认列表，如果接口数据中存在对应的 wanIndex，则使用接口数据，否则使用默认空值
				var mergedList = defaultList.map(function(defaultItem){
					var wanIndex = defaultItem.wanIndex;
					// 如果接口数据中有这个 wanIndex，使用接口数据
					if(apiDataMap[wanIndex]){
						return apiDataMap[wanIndex];
					}
					// 否则使用默认空值
					return defaultItem;
				});
				
				return mergedList;
			},
			getBandwidthMap(){
				return {
					'n25': '5MHz',
					'n50': '10MHz',
					'n75': '15MHz',
					'n100': '20MHz'
				};
			},
			//校验子网掩码
            isMask(str){
                var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/;
                return exp.test(str);
            },
			//校验IP
            isValidIP(ip){
                var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/
                return reg.test(ip);
            },
			customizedAddClick(){
				var vm = this, id = 'id',
					params={
						custParamName: '',
						custParamValue: '',
						custParamPath: '',
					};
				if(vm.addOrEditForm.custParam.length == 0){
					params[id] = '1';
				}else{
					var idList=[];
					vm.addOrEditForm.custParam.map((item)=>{
						idList.push(item.id + '');
					})
					params.id = vm.createId(1,idList); 
					var delNum = 0;
					vm.addOrEditForm.custParam.map((item)=>{
						delNum+=1;
					})
				}
				vm.addOrEditForm.custParam.push(params);
				event.stopPropagation();
			},
			customizedDelClick(row){
				var vm = this;
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					vm.addOrEditForm.custParam.map(function(item,index){
						if(item.id == row.id){
							vm.addOrEditForm.custParam = vm.addOrEditForm.custParam.filter((items)=>{
								return items.id != row.id
							})
						}
					})
				}).catch(()=>{})
			},
			removeGroupItem(code) {
				var vm = this, idx = vm.advanceTreeSelected.indexOf(code);
				if(idx > -1) {
					vm.advanceTreeSelected.splice(idx,1);
					vm.$refs.advanceTree.setChecked(code,false);
					vm.advanceTreeConfirm();
				}
			},
			smhd(val) {
				var vm = this, reg = /^\d+[s|m|h|d]{1}$/;
				return reg.test(val);
			},
			noZh(val) {
				var vm = this, no_zh = /^(?:(?![\u4E00-\u9FA5]|[\uFE30-\uFFA0]).)+$/;
				return no_zh.test(val);
			},
			addKey(idx) {
				var vm = this, key = 'index_'+idx;
				if(vm.extend[key] == undefined) {
					vm.$set(vm.extend, key, false);
				} 
			},
			addIpsecItem() {
				var vm = this,
					list = vm.advanceForm.ipsecList.map(function(item){
						return item.IPSEC_INDEX - 0;
					}), 
					max = 1, idx = 0;
				if(list.length) {
					max = Math.max.apply(Math, list) + 1;
					for(var i=0; i<=max; i++) {
						if(!list.includes(i)) {
							idx = i;
							break;
						}
					}
				}
				vm.advanceForm.ipsecList.push({
					TUNNEL_ENABLE: '0',
					IPSEC_INDEX: idx,
					AUTHBY: '',
					LEFT_AUTH: '',
					RIGHT_AUTH: '',
					TUNNEL_GATEWAY: '',
					LEFT_SUBNET: '',
					RIGHT_SUBNET: '',
					LEFT_IDENTIFIER: '',
					RIGHT_IDENTIFIER: '',
					LEFT_CERT: '',
					SECRET_KEY: '',
					RIGHT_SECRET_KEY: '',
					LEFTSOURCEIP: '',
					IKE_ENCRYPTION: '',
					IKE_DH_GROUP: '',
					IKE_AUTHENTICATION: '',
					IKELIFETIME: '',
					ESP_ENCRYPTION: '',
					ESP_DH_GROUP: '',
					ESP_AUTHENTICATION: '',
					KEYLIFE: '',
					REKEYMARGIN: '',
					DPDACTION: '',
					DPDDELAY: ''
				});
				vm.$nextTick(function(){
					vm.extend['index_'+idx] = true;
				});
			},
			initParamKeys() {
				var vm = this,
					selected = vm.productList.filter(function(item){
						return item.value == vm.addOrEditForm.productType;
					})[0],
					type = selected?selected.name:'',
					allProps = [
						'advanceEnable',
						'ipsec_enable',
						'TUNNEL_ENABLE',
						'IPSEC_INDEX',
						'AUTHBY',
						'LEFT_AUTH',
						'RIGHT_AUTH',
						'TUNNEL_GATEWAY',
						'LEFT_SUBNET',
						'RIGHT_SUBNET',
						'LEFT_IDENTIFIER',
						'RIGHT_IDENTIFIER',
						'LEFT_CERT',
						'SECRET_KEY',
						'RIGHT_SECRET_KEY',
						'LEFTSOURCEIP',
						'IKE_ENCRYPTION',
						'IKE_DH_GROUP',
						'IKE_AUTHENTICATION',
						'IKELIFETIME',
						'ESP_ENCRYPTION',
						'ESP_DH_GROUP',
						'ESP_AUTHENTICATION',
						'KEYLIFE',
						'REKEYMARGIN',
						'DPDACTION',
						'DPDDELAY',
						'mtu',
						'totalTxPower',
						'powerRamping',
						'preambleInitTargetPower',
						'poNominalPusch',
						'poNominalPucch',
						'alpha',
						'targetUlSinr',
						'powerPA',
						'powerPB',
						'dnsHostName',
						'dnsTimeZone',
						'dnsAddress1',
						'dnsAddress2',
						'dnsAddress3',
						'ntpEnable',
						'ntpPort1',
						'ntpServer1',
						'ntpPort2',
						'ntpServer2',
						'ntpPort3',
						'ntpServer3'
					],
					ignores = {
						'RTS': ['LEFT_AUTH','RIGHT_AUTH','LEFT_CERT','SECRET_KEY','LEFT_SUBNET',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3'],
						'RTD': ['LEFT_AUTH','RIGHT_AUTH','LEFT_CERT','SECRET_KEY','LEFT_SUBNET',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3'],
						'QRTB': ['LEFT_CERT','RIGHT_SECRET_KEY','LEFT_SUBNET',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3',
							'ntpPort2',
							'ntpPort3'
						],
						'BAIBLQ': ['LEFT_CERT','RIGHT_SECRET_KEY','LEFT_SUBNET',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3',
							'ntpPort2',
							'ntpPort3'
						],
						'BLX': ['LEFT_CERT','RIGHT_SECRET_KEY','LEFT_SUBNET',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3',
							'ntpPort2',
							'ntpPort3'
						],
						'MLQ': ['LEFT_CERT','RIGHT_SECRET_KEY','LEFT_SUBNET',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3',
							'ntpPort2',
							'ntpPort3'
                        ],
						'QAFA': ['LEFTSOURCEIP','ntpPort2','ntpPort3'],
						'QATA': ['LEFTSOURCEIP','ntpPort2','ntpPort3'],
						'QAFB': ['LEFTSOURCEIP','LEFT_SUBNET','ntpPort2','ntpPort3'],
						'PM-B4860': [
							'LEFTSOURCEIP',
							'totalTxPower',
							'powerRamping',
							'preambleInitTargetPower',
							'poNominalPusch',
							'poNominalPucch',
							'alpha',
							'targetUlSinr',
							'powerPA',
							'powerPB',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3',
							'ntpPort2',
							'ntpPort3'
						],
						'CR-B4860/BU': [
							'LEFTSOURCEIP',
							'totalTxPower',
							'powerRamping',
							'preambleInitTargetPower',
							'poNominalPusch',
							'poNominalPucch',
							'alpha',
							'targetUlSinr',
							'powerPA',
							'powerPB',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3',
							'ntpPort2',
							'ntpPort3'
						],
						'CR-B4860/EU': [
							'LEFTSOURCEIP',
							'totalTxPower',
							'powerRamping',
							'preambleInitTargetPower',
							'poNominalPusch',
							'poNominalPucch',
							'alpha',
							'targetUlSinr',
							'powerPA',
							'powerPB',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3',
							'ntpPort2',
							'ntpPort3'
						],
						'CR-B4860/RU': [
							'LEFTSOURCEIP',
							'totalTxPower',
							'powerRamping',
							'preambleInitTargetPower',
							'poNominalPusch',
							'poNominalPucch',
							'alpha',
							'targetUlSinr',
							'powerPA',
							'powerPB',
							'dnsHostName',
							'dnsTimeZone',
							'dnsAddress1',
							'dnsAddress2',
							'dnsAddress3',
							'ntpPort2',
							'ntpPort3'
						]
					}; 
				vm.paramKeys = [];
				vm.advanceTree = [];
				vm.advanceTreeSelected = [];
				vm.defaultRreeChecked = [];
				if(ignores[type]) {
					vm.paramKeys = allProps.filter(function(prop){
						return !ignores[type].includes(prop);
					});
				}
				if(vm.groupShow.wan) {
					vm.advanceTree.push({
						id: 'wan',
						label: 'WAN',
						children: [
							{ id: 'wanConfig', label: 'WAN Config' }
						]
					});
				}
				if(vm.groupShow.ipsec) {
					vm.advanceTree.push({
						id: 'ipsec',
						label: 'IPSec',
						children: [
							{ id: 'ipsecSettings', label: 'IPSec Settings' }
						]
					})
				}
				if(vm.groupShow.power) {
					vm.advanceTree.push({	
						id: 'power',
						label: 'Power Control',
						children: [
							{ id: 'powerControl', label: 'Power Control Parameters' }
						]
					})
				}
				var system = {
					id: 'system',
					label: 'System',
					children: []
				};
				if(vm.groupShow.dns) {
					system.children.push({ id: 'dns', label: 'DNS' })
				}
				if(vm.groupShow.ntp) {
					system.children.push({ id: 'ntp', label: 'NTP' })
				}
				if(vm.groupShow.dns || vm.groupShow.ntp) {
					vm.advanceTree.push(system);
				}
			},
			removeIpsec(index) {
				var vm = this;
				vm.advanceForm.ipsecList.splice(index,1);
			},
			advanceSettingClick() {
				var vm = this;
				vm.advanceTreeShow = true;
				vm.modelTypeAddShow = false;
			},
			advanceTreeConfirm() {
				var vm = this, checkedKeys = '';
				checkedKeys = vm.$refs.advanceTree.getCheckedKeys();
				vm.advanceTreeSelected = checkedKeys;
				vm.advanceSettingCancel();
			},
        	advanceSettingCancel(){
        		var vm = this;
				vm.$refs.advance.resetFields();
				vm.advanceTreeShow = false;
        	},
            //产品类型改变
			productTypeChange(type) {
				var vm = this;
				//目标版本清空，需根据产品类型重新加载对应目标版本及初始版本
				vm.addOrEditForm.originalVersion = '';
				vm.resultOriginalVersionList = [];
				vm.addOrEditForm.targetVersion = '';
				vm.tarVersionList = [];
				vm.commonOriginalVerList();
				vm.initParamKeys();
				vm.$refs.advance.resetFields();
			},
			//详情或修改，获取详情信息
            getInfoById(id) {
	            var vm = this,
	                params = {
	                    policyId: id
	                },
	                url = '${ctx}/SON/SelfConfiguration/getSelfConfigurationInfo.action';
	
	            // 获取策略信息
	            axios.post(url, stringify(params)).then(function(res){
					var data = res.data;
					//回显对应的产品类型 value 对应的 name
					vm.curProductName = data.platform;	
	                Object.assign(vm.addOrEditForm, data);
					
					if(data.halob_enable == "1" || data.halob_enable == '' || data.halob_enable == null){//打开
						vm.showMMEMain = false;
					}else{//关闭
						vm.showMMEMain = true;
					}
					if(data.mme_ip == "" || data.mme_ip == null){
						vm.addOrEditForm.mmeGroup = [];
					}else{
						vm.addOrEditForm.mmeGroup = data.mme_ip.split(",")
					}
					vm.addOrEditForm.mmeStr = data.mme_ip;
					
					if(vm.curProductName == "DXDF"){
						if(data.frequency == "" || data.frequency == null){
							vm.addOrEditForm.frequency = '';
						}else{
							vm.addOrEditForm.frequency = data.frequency;
						}
					}else{
						if(data.frequency != null){
							if(translateToFre(data.frequency) == false){
								
							}else{
								vm.addOrEditForm.frequency = translateToFre(data.frequency)
							}
						}
					}
					if(data.autoConfigMode == "" || data.autoConfigMode == null){
						vm.addOrEditForm.autoConfigMode = '0';
					}else{
						if(data.autoConfigMode == 'MAP'){
							vm.addOrEditForm.autoConfigMode = '0';
						}else if(data.autoConfigMode == 'TAC'){
							vm.addOrEditForm.autoConfigMode = '1';
						}else if(data.autoConfigMode == 'SN'){
							vm.addOrEditForm.autoConfigMode = '2';
						}
					}
					if(vm.curProductName == "NBIOT" && data.ul_frequency != null){
						if(translateToFre(data.ul_frequency) == false){
							
						}else{
							vm.addOrEditForm.ul_frequency = translateToFre(data.ul_frequency)
						}
					}
					if(data.txPower == "" || data.txPower == null){
						vm.addOrEditForm.txPower = [];
					}else{
						vm.addOrEditForm.txPower = data.txPower.split(",")
					}

					if(data.subframe_assignment != null){
						vm.addOrEditForm.subframe_assignment = data.subframe_assignment;
					}
					if(data.special_subframe_patterns != null){
						vm.addOrEditForm.special_subframe_patterns = data.special_subframe_patterns;
					}
	                //回显初始版本
	                vm.addVersionBtnShow = false;
	                vm.resultOriginalVersionShow = true;
	                if(data.originalVersion != null){
	                	if(data.originalVersion == 'all'){
	                		vm.addOrEditForm.specifyVersionType = '1';
		               	}else{
		               		var originlist =  data.originalVersion,
		               		resultOriginlist = originlist.split(',');
		               		vm.resultOriginalVersionList = resultOriginlist.map(function(item){ return {originalVersion: item};});		               
	                	}
	                }
					// 高级参数回显
					for(key in vm.advanceForm) {
						vm.advanceForm[key] = data[key];
					}
					// init advance params
					vm.initParamKeys(); 
					vm.advanceTreeSelected = data.showGroup?data.showGroup.split(','):[];
					vm.defaultRreeChecked = data.showGroup?data.showGroup.split(','):[];
				});
	        },       
        	//software upgrade,license,parameter config 三个模块切换时，使其打开的右侧内容关闭； 需校验内容是否发生变化
        	selectMethodChange(val){
        		var vm = this;
        		vm.commonRightModelClose();
				vm.commonParamConfigContent();
				if(val == '2'){
					vm.$refs.ctablePlan.refresh();
					if(vm.curProductName == "QAFA" || vm.curProductName == "QATA" ){
						vm.$refs.ctableMap.refresh();
						vm.$refs.enbIdTactable.refresh();
						vm.$refs.snCtableMap.refresh();
					}
				}
        	},
        	//回显参数配置模块内容
        	commonParamConfigContent(){
        		var vm = this;
        		vm.commonParamsMethod();
				if(vm.functionModulesSelect == '2'){
					//根据产品类型回显对应基本信息
					if(vm.curProductName === '' || vm.curProductName === null || vm.curProductName === undefined){
						vm.curProductName = (vm.productList[0] || []).name;
					}				
				}
        	},
        	//parameter config 中的 parameter data pool, batch import, region deploying 三个模块切换时，使其打开的右侧内容关闭； 需校验内容是否发生变化
        	clickTab(val){
        		var vm = this;
				vm.parameterConfigActive == 'paramTwo' ? vm.urlConfigPlan = '${ctx}/SON/SelfConfiguration/getSelfConfigPlanningPageList.action' : vm.urlConfigPlan = '';
        		vm.commonRightModelClose();
			},
			//右侧弹窗内容关闭
			commonRightModelClose(){
				var vm = this;
        		vm.addVersionShow = false;
        		vm.licenseImportShow = false;
        		vm.modelTypeAddShow = false;
        		vm.specifyImportShow = false;
        		vm.specifyModifyShow = false;
        		vm.specifyInfoShow = false;
				vm.advanceTreeShow = false;
        		vm.coordinateTacImportShow = false;
        		vm.coordinateTacModifyShow = false;
				vm.enbIdTacModifyShow = false;
				vm.snAddShow = false;
			},
        	//--------------------------------------------software upgrade
        	//新增版本按钮点击事件
			addVersionClick(){
				var vm = this;
				vm.addVersionShow = true;
				vm.curSelectVersionData = [];
				vm.softwareAddVersionForm.versionList = [];
				vm.$refs.softwareOriginalVersionTable.clearSelection();
				vm.commonOriginalVerList();
			},
			//根据已选产品类型加载对应的初始版本数据
			commonOriginalVerList(){
				var vm = this;
                axios.post("${ctx}/plugandplay/policy/getOriginalVersionPageList.action",stringify({
                	product: vm.addOrEditForm.productType,
                    searchText: '',
                    page: 1,
                    rows: 50
                })).then(function(res){
					var data = res.data;
					vm.softwareOriginalVersionList = data.rows || [];
				});
			},
			//手动添加 version
			addVersionBtn(){
				var vm = this, versionText = vm.softwareAddVersionForm.versionStr, str = '';
				if(versionText == ''){
					vm.versionErrorMessage = '<%=rb.getString("ShuRuBanBenHao")%>';
				}else{
					if(versionText){
						str = versionText;
						if(vm.softwareAddVersionForm.versionList.indexOf(str) == -1){
							vm.softwareAddVersionForm.versionList.push(str);
							vm.softwareAddVersionForm.versionStr = '';
							vm.versionErrorMessage = '';
							vm.versionMessage = '';
							vm.$refs.softwareAddVersionForm.validateField('itemTest');
						}else{
							vm.versionErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.versionErrorMessage = '<%=rb.getString("ShuRuBanBenHao")%>';
					}
				}
			},
			//删除version
			removeVersion(item){
				var vm = this, index = vm.softwareAddVersionForm.versionList.indexOf(item);
				if(index !== -1){
					vm.softwareAddVersionForm.versionList.splice(index,1)
				}
				vm.versionErrorMessage = '';
			},
			//列表选中
			versionBatchSelect(selection){
				var vm = this; 
				vm.curSelectVersionData = selection;
			},
        	//add version submit
        	addVersionSubmit(){
        		var vm = this;
        		var manualVersionList = vm.softwareAddVersionForm.versionList;
        		var versionList = vm.curSelectVersionData.map(function(item){ return item.originalVersion});
        		//手动添加版本号（还未在OMC上线或上传的版本号）和已上线OMC 的版本号二选其一皆可
        		if(manualVersionList.length == 0 && vm.curSelectVersionData.length == 0){
        			vm.versionMessage = '<%=rb.getString("ZhiShaoXuanZeYiZhongBanBenFangShi")%>';
        			return;
        		}else{        		
        			vm.versionMessage = '';
        			//数组合并 去重
            		var mergeVersionList = [];
            		var mergelist = manualVersionList.concat(versionList);
            		for(var i =0, len = mergelist.length; i<len; i++){
            			if(mergeVersionList.indexOf(mergelist[i]) === -1){
            				mergeVersionList.push(mergelist[i])
            			}
            		}
            		var curList = mergeVersionList.map(function(item){
						return {originalVersion: item }
					});
            		if(vm.resultOriginalVersionList.length > 0){
            			//多次添加时，需将新增及已有的版本号合并去重
            			var length1 = vm.resultOriginalVersionList.length;
            			var length2 = curList.length;
            			for (var i = 0; i<length1; i++){
            				for(var j= 0; j<length2; j++){
            					if(vm.resultOriginalVersionList.length > 0){
            						if(vm.resultOriginalVersionList[i]['originalVersion'] === curList[j]['originalVersion']){
            							vm.resultOriginalVersionList.splice(i,1);
            							length1 --;
            						}
            					}
            				}
            			}
            			for(var n = 0; n< curList.length; n++){
            				vm.resultOriginalVersionList.push(curList[n]);
            			}
            		}else{
            			vm.resultOriginalVersionList = mergeVersionList.map(function(item){
							return {originalVersion: item }
						})
            		}
            		vm.addOrEditForm.originalVersion = vm.resultOriginalVersionList.map(function(item){ return item.originalVersion;}).join(',');
            		vm.addVersionShow = false;
            		vm.addVersionBtnShow = false;
                    vm.resultOriginalVersionShow = true;
        		}        		
        	},
        	//add version cancel
        	addVersionCancel(){
        		var vm = this;
        		vm.addVersionShow = false;
        		vm.curSelectVersionData = [];
        		vm.$refs.softwareAddVersionForm.resetFields();
        		vm.softwareAddVersionForm.versionList = [];
        		vm.$refs.softwareOriginalVersionTable.clearSelection();
        	},
        	//从结果版本列表中删除
        	deleteVersionItem(item){
				var vm = this;
				var index = vm.resultOriginalVersionList.indexOf(item);
				if(index !== -1){
					vm.resultOriginalVersionList.splice(index,1)
				}
			},
			//clear 操作
			clearVersionBtnClick(){
				this.resultOriginalVersionList = [];
			},
       		//------------------------------------------------license			
       		//license delete
       		deleteLicenseClick(row,evt){
				var vm = this, params = { fileNames : row.file_name }
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/cell/license/doClearLicenseFile.action",stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					})
	    					vm.$refs.licenseTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()	
				vm.licenseImportShow = false;
			},
			//导入
			importLicenseClick(){
        		var vm = this;
				vm.licenseImportShow = true;
			},
			//文件上传成功函数 
			moreCheckFile(res,file){    //发送请求，校验device文件内容 
				var vm = this;
				if(res.success){
					if(res.suc_count>0){
						vm.$message({
							type: 'success',
							message: '<%=rb.getString("ChengGong")%>'
						});
					}else {
						vm.$message({
							type: 'warning',
							message: '<%=rb.getString("ShiBai")%>'
						});
					}
					vm.licenseImportShow = false;
					vm.$refs.licenseTable.refresh();
					vm.moreCloseFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				//修改已选择文件状态  
				var fileList = vm.$refs.moreUpload.uploadFile;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},	    	
			//选择文件后，校验格式，并赋值页面显示 
			moreFileChange(file,fileList){ 
				var vm = this;
				var fileIndex = file.name.lastIndexOf(".");
                var fileType = file.name.substr(fileIndex + 1, file.name.length);
                if(['lic'].indexOf(fileType.toLowerCase()) === -1){
                    return false;
                }else{
                    vm.fileName += file.name+',';
                    vm.moreFileParams.FileName = file.name;
                }
			},
			// 选择文件
			moreFileSelect(){  
				var vm =this;
	    	    vm.fileName = '';
				vm.$refs.moreUpload.clearFiles();
				vm.$refs['file_up'].click();
			},
			// 移除导入文件
			moreCloseFileSelect(){
				var vm = this;
				vm.fileName = '';			
				vm.$refs.moreUpload.clearFiles();
			},
			//提交时匹配的文件及文件流
			moreFileRequest(file){
				var vm = this;
				vm.fileData.push(file.file);
			},
			/*确定导入*/
	        licenseImportSubmit() {
				var vm = this, fileImportArr = [], productType = vm.addOrEditForm.productType;
				vm.$refs.licenseForm.validate((valid) => {
	                if (valid){ 
	   					var fd = new FormData(),
	    					config = {
	    						headers: { 'Content-Type': 'multipart/form-data' }
	    					};
	                    vm.$refs.moreUpload.submit();
	    				vm.uploadBtnDisabled = true;
	    				vm.licenseLoading = true;
	    				if(vm.fileData.length > 0){
	    					vm.fileData.forEach(file =>{
	    					    //只有 lic 格式可行
	    					    if(file.name.substr(file.name.lastIndexOf(".")) === '.lic'){
                                   var obj = {
                                        fileName: file.name,
                                        fielSize: file.size
                                   }
                                   fileImportArr.push(obj)
                                   fd.append('uploadFile',file);
                                }
		    				})
	    				}
	    				fd.append('fileArr',JSON.stringify(fileImportArr));//文件名
	    				fd.append('productType',productType);
	    				axios.post("${ctx}/cell/license/uploadLicenseFile.action",fd,config).then(function(res){
	    					if(res.data["success"]){
	    						vm.$message.success('<%=rb.getString("DaoRuChengGong")%>');
	    					    vm.$refs.licenseTable.refresh();
	    					}else{
	    						vm.$message.error(res.data["message"]);
	    					}
	    					vm.licenseImportCancel();
	    				})
	               	}
	           	})
			},
			// 关闭导入弹出框
			licenseImportCancel(){
				var vm = this;
				vm.licenseImportShow = false;
				vm.uploadBtnDisabled = false;
				vm.licenseLoading = false;
				vm.fileList = [];
				vm.fileName = '';
				vm.$refs.licenseForm.resetFields();
				vm.importModeSelection = 'moreFile';
				vm.fileData =[];
			},
			//--------------------------------Parameters Data Pool
			//  add mudule btn click
			addModuleBtnClick(){
				var vm = this;
				vm.modelTypeAddShow = true;
				vm.moduleErrorMessage = '';
				vm.advanceTreeShow = false;
				vm.moduleTitle = '<%=rb.getString("TianJia")%>';
				vm.moduleFlag = 'add';
				vm.moduleVal = '';
				vm.$refs.moduleFm.resetFields();
			},	
			//module table: modify
			moduleModifyClick(row, index){
				var vm = this,
					params = {
						module_type: row.module_type?row.module_type.split(','):[],
						band_width: row.band_width,
						frequency: row.frequency	
					};
				vm.modelTypeAddShow = true;
				vm.moduleErrorMessage = '';
				vm.advanceTreeShow = false;
				try{
					vm.$refs.moduleFm.resetFields();
				}catch(e){}
				vm.moduleVal = '';
				vm.moduleTitle = '<%=rb.getString("XiuGai")%>';
				vm.moduleFlag = 'edit';
				vm.moduleIndex = index;
				Object.assign(vm.moduleForm, params);
			},
			//module table: delete
			moduleDeleteClick(row, index) {
				var vm = this;
				vm.$confirm('<%= rb.getString("QueRenShanChu")%>','<%= rb.getString("QueRen")%>').then(function(r){
                    if(r) {
                    	vm.addOrEditForm.moduleTypeList.splice(index,1);
                    }
                }).catch(function(){});
				vm.modelTypeAddShow = false;
			},
			//add module dialog：  click the add btn to save
			addModuleType() {
				var vm = this, versionText = vm.moduleVal.trim(), str = '';
				if(versionText == ''){				
					vm.moduleErrorMessage = '<%=rb.getString("BiTian")%>';
				}else{
					if(versionText){
						str = versionText;
						if(vm.moduleForm.module_type.indexOf(str) == -1){
							vm.moduleForm.module_type.push(str);
							vm.moduleVal = '';
							vm.moduleErrorMessage = '';
						}else{
							vm.moduleErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.moduleErrorMessage = '<%=rb.getString("BiTian")%>';
					}
				}
			},	
        	//add module dialog： delete 
			removeModule(idx) {
				var vm = this;
				vm.moduleForm.module_type.splice(idx,1);
			},
			//save module type 
        	modelTypeAddSubmit(){
				var vm = this, flag = false,
					row = {
						module_type: vm.moduleForm.module_type.join(','),
						band_width: vm.moduleForm.band_width,
						frequency: vm.moduleForm.frequency
					};
				if(vm.moduleForm.module_type.length == 0){
					vm.moduleErrorMessage ='<%=rb.getString("BiTian")%>';
					flag = true;
				}
				vm.$refs.moduleFm.validate(function(valid){
					if(valid && flag == false){
						if(vm.moduleFlag == 'add') {
							vm.addOrEditForm.moduleTypeList.push(row);
						}else {
							Object.assign(vm.addOrEditForm.moduleTypeList[vm.moduleIndex], row);
						}
						vm.modelTypeAddCancel();
					}
				});
        	},
        	//close add module dialog
        	modelTypeAddCancel(){
        		var vm = this;
				vm.modelTypeAddShow = false;
				vm.moduleErrorMessage = '';
				vm.$refs.moduleFm.resetFields();
        	},
			//add mme ip
			addMMEOne(){
				var vm = this, value = vm.addOrEditForm.mme, str = '', msg = '<%=rb.getString("IPDiZhi")%>';
				if(value == ''){
					vm.errorMsgOne = msg;
				}else{
					if(isValidIP(value)){
						str = value;
						if(vm.addOrEditForm.mmeGroup.indexOf(str) == -1){
							vm.addOrEditForm.mmeGroup.push(str);
							vm.addOrEditForm.mme = '';
							vm.errorMsg = '';
						}else{
							vm.errorMsgOne = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.errorMsgOne = msg;
					}
				}
			},
			//delete mme ip
			removeMMEOne(item){
				var vm = this;
				var index = vm.addOrEditForm.mmeGroup.indexOf(item);
				if(index !== -1){
					vm.addOrEditForm.mmeGroup.splice(index,1)
				}
				vm.errorMsgOne = '';
			},
			// 频率转换
            changeToFrequencyOne(type) { 
				if(type == 'dl'){
					var value = this.addOrEditForm.frequency;
					if(value == "");
					else this.addOrEditForm.frequency = translateToFre(this.addOrEditForm.frequency);
				}else if(type == 'ul'){
					var value = this.addOrEditForm.ul_frequency;
					if(value == "");
					else this.addOrEditForm.ul_frequency = translateToFre(this.addOrEditForm.ul_frequency);
				}
            },
			// 频率还原
			changeToEarfcnOne(type){
				if(type == 'dl'){
					var value = this.addOrEditForm.frequency;
					var showFlag = value.indexOf("(");
					if(showFlag == -1);
					else this.addOrEditForm.frequency = this.addOrEditForm.frequency.substring(0,showFlag);
				}else if(type == 'ul'){
					var value = this.addOrEditForm.ul_frequency;
					var showFlag = value.indexOf("(");
					if(showFlag == -1);
					else this.addOrEditForm.ul_frequency = this.addOrEditForm.ul_frequency.substring(0,showFlag);
				}
			},
        	//--------------------------batch import      	
        	//导入
        	specifyImportClick(){
               	var vm = this;
   				vm.specifyImportShow = true;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.enbIdTacModifyShow = false;
				vm.coordinateTacImportShow = false;
				vm.coordinateTacModifyShow = false;
				vm.snAddShow = false;
			},
			importFile(){
				$("#uploadForm_configPlan input[name='uploadFile']").click();
			},
			exportTemp(id){
				var params = { tempType:id };
				exportByForm("${ctx}/SON/SelfConfiguration/downloadSelfConfigTemp.action", params);
			},
			handerCloseExport(){
				this.exportTypeFlag = false;
			},
			specifyImportSubmit(){
               	var vm = this;
				vm.$refs.importForm.validate((valid) => {
					if(valid){
						vm.importLoading = true;
						vm.importSpecialLoading = true;
						var files = document.querySelector("#uploadFileConfigPlan").files;
						$("#uploadForm_configPlan [name=fileSize]").val(files[0].size);
				    	$("#uploadForm_configPlan [name=operType]").val(vm.importForm.importType);
				    	if(vm.curType == 'modify'){
							$("#uploadForm_configPlan [name=policyId]").val(vm.curPolicyId);
						}
						uploadWithProgress({
							url: "${ctx}/SON/SelfConfiguration/importSelfConfigPlanningInfos.action",
							form: document.querySelector("#uploadForm_configPlan"),
							success: function (data) {
						     	if(data["success"]){
						     		$("#uploadForm_configPlan input[name='uploadFile']").val("");
						     		vm.$message({
			    						message:"<%=rb.getString("ChengGong")%>",
			    						type:'success'
			    					})
						     		vm.$refs.ctablePlan.refresh();						     		
						     	}else{
						     		if(data["msg"] == "1"){
						     			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>');
						     		}else if(data["msg"] == "2"){
						     			$("#uploadForm_configPlan input[name='uploadFile']").val("");
						     			vm.$refs.ctablePlan.refresh();
						     			vm.downloadVisible = true;
					        		}else{
					        			vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
					        		}
						     	}
						     	vm.specifyImportCancel();
						     }
						});
					}
				})
			},
			closeDownloadDialog(){
				var vm = this;
				vm.downloadVisible = false;
			},
			downloadErrorFile(){
				var url = "${ctx}/SON/SelfConfiguration/downloadFailureFile.action"
				exportByForm(url,{});
				this.downloadVisible = false;
			},
			specifyImportCancel(){
               	var vm = this;
               	vm.specifyImportShow = false;
               	vm.importLoading = false;
				vm.importSpecialLoading = false;
				vm.importForm = {
					importType: "0",
					filePath: ""
				}
				$("#uploadForm_configPlan input[name='uploadFile']").val("")
				vm.$refs.importForm.resetFields();
			},
        	//导出
        	specifyExportClick(){
				var vm = this,
					params = {
					search_text : this.params_plan.searchText,
					timeZone : timeZone
				}
				if(vm.curType == 'modify'){
					params.policyId = vm.curPolicyId;
				}
				var url = "${ctx}/SON/SelfConfiguration/downloadSelfConfigPlanningResList.action"
				exportByForm(url,params);
			},
			//--------------------------------batch import
			showCell2Param(){
				var vm = this;
				event.stopPropagation();
				vm.showCell2 = true;
			},
			//修改
			specifyModifyClick(row, evt){
				var vm = this;
				vm.specifyModifyShow = true;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				//regional
				vm.coordinateTacImportShow = false;
        		vm.coordinateTacModifyShow = false;
				vm.enbIdTacModifyShow = false;
				vm.snAddShow = false;
				vm.curSpecifyModifyRow = row;
				vm.rowDataPlan = row;
				//回显数据  k公共部分可作为公共方法阴影
				vm.platform = row.platform;	
				vm.commonParamsMethod();
				if(vm.platform == "intel"){
					vm.showMode = true;
					vm.showItem = true;
					vm.showBasic = true;
					vm.shownbi = true;
					vm.mapFlag = false;
				}else if(vm.platform == "qa"){
					vm.mapFlag = true;
					vm.showBasic = true;
					vm.shownbi = true;
					vm.showMode = false;
					vm.showItem = false;
				}else if(vm.platform == 'NBIOT'){
					vm.showMME = true;
					vm.shownbi = false;
					vm.showBasic = false;
					vm.showMode = false;
					vm.mapFlag = false;
					vm.showItem = false;
					vm.frequencyLabel = 'DL Frequency'
				}
				var params = {
					id : row.id
				}
				axios.post("${ctx}/SON/SelfConfiguration/getSelfConfigPlanningInfos.action",stringify(params)).then(function(response){
					var data = response.data;
					vm.paramList = data.paramList;
					//如果SAS开关打开，则该参数不可配置
			    	if(SASEnble == "1"){
			    		vm.sasDisabled = true;
			    	}
			    	if(data == "" || data == null){
						
					}else{
						vm.basicForm.bands_support = data.bands_support;
						//vm.basicForm.band_width = data.band_width;
						if(vm.curProductName == "DXDF"){
							if(data.frequency == "" || data.frequency == null){
								vm.basicForm.frequency = '';
							}else{
								vm.basicForm.frequency = data.frequency;
							}
							if(data.band_width == "" || data.band_width == null){
								vm.basicForm.band_width = '';
							}else{
								vm.basicForm.band_width = data.band_width;
							}
						}else{
							if(data.frequency != null){
								if(translateToFre(data.frequency) == false){
									
								}else{
									vm.basicForm.frequency = translateToFre(data.frequency)
								}
							}
							if(data.band_width != null){
								vm.basicForm.band_width = data.band_width;
							}
						}
						if(vm.platform == 'NBIOT' && data.ul_frequency != null){
							if(translateToFre(data.ul_frequency) == false){
								
							}else{
								vm.basicForm.ul_frequency = translateToFre(data.ul_frequency)
							}
						}
						if(data.subframe_assignment != null){
							vm.basicForm.subframe_assignment = data.subframe_assignment;
						}
						if(data.special_subframe_patterns != null){
							vm.basicForm.special_subframe_patterns = data.special_subframe_patterns;
						}
						vm.basicForm.plmn_id = data.plmn_id;
						vm.basicForm.tac = data.tac;
						vm.basicForm.cell_identity = data.cell_identity;
						vm.basicForm.phycellid = data.phycellid;
						vm.basicForm.root_sequence_index = data.root_sequence_index;
						if(data.mme_ip == "" || data.mme_ip == null){
							vm.basicForm.mmeGroup = [];
						}else{
							vm.basicForm.mmeGroup = data.mme_ip.split(",")
						}
						vm.basicForm.mmeStr = data.mme_ip;
						vm.basicForm.carrier_mode = data.carrier_mode;
						vm.basicForm.halob_enable = data.halob_enable;
						vm.basicForm.timeZoneUtc = data.timeZoneUtc;
						if(data.txPower == '' || data.txPower == null){
							vm.basicForm.txPower = '';
							vm.basicForm.dxdfTxPower = [];
						}else{
							vm.basicForm.txPower = parseInt(data.txPower);
							vm.basicForm.dxdfTxPower = data.txPower.split(",");
						}
						vm.settingForm.ipsec_switch = data["ipsec_switch"]==0?false:true;
						vm.settingForm.ipsec_enable = data.ipsec_enable;
						vm.settingForm.ipsec_rightikeport = data.ipsec_rightikeport;
						vm.settingForm.left_interface = data.left_interface;
						vm.settingForm.ipsecList = data.ipsecList;
						if(data.ipsecList && data.ipsecList.length !=0){
							data.ipsecList.map(function(item){
								vm.settingForm.defaultIpsecList.push(item)
							})
						}
						//wan config 'CR-B4860' || 'BAIBLQ'
						if(vm.curProductName == 'CR-B4860' || vm.curProductName == 'BAIBLQ'){
							vm.wanConfigForm.wanSendEnable = data.wanSendEnable || '0';
							// 使用 mergeWanBindingList 方法合并接口数据和默认列表
							// 确保所有 wanIndex (1-4) 都存在，缺失的显示为空值
							vm.wanConfigForm.wanBindingList = vm.mergeWanBindingList(data.wanBindingList);
							vm.wanConfigForm.originalWanBindingList = JSON.parse(JSON.stringify(vm.wanConfigForm.wanBindingList));
						}
						//打开
						if(data.halob_enable == "1"){
							vm.showMME = false;
							vm.showSetting = false;
						}else{
							//关闭
							vm.showMME = true;
							vm.showSetting = vm.platform == 'NBIOT' ? false : true;
						}
						//4860 判断 data 中 是否含有 cell2params
						if(data.cell2params && (vm.curProductName == 'CR-B4860' || vm.curProductName == 'MLN')){
							vm.showCell2 = true;
							vm.basicForm.cellPlmnId = data.cell2params.plmn_id;
							vm.basicForm.cellBandwidth = data.cell2params.band_width; //带宽
							vm.basicForm.cellEci = data.cell2params.cell_identity; // eci
							vm.basicForm.cellPhycellid = data.cell2params.phycellid; //pci
							vm.basicForm.cellRootIndex = data.cell2params.root_sequence_index;
							vm.cellMmeStr = data.cell2params.mme_ip;
							if( data.cell2params.mme_ip){
								vm.basicForm.cellMmeGroup = data.cell2params.mme_ip.split(",");
							}
							if(data.cell2params.frequency != null){
								if(translateToFre(data.cell2params.frequency) == false){
									
								}else{
									vm.basicForm.cellFrequency = translateToFre(data.cell2params.frequency)
								}
							}
						}
						setTimeout(function(){
							initForm(vm.$refs.basicForm);
							if(vm.showSetting && vm.curProductName != "DXDF"){
								initForm(vm.$refs.settingForm);
							}	
							if(vm.curProductName == 'CR-B4860' || vm.curProductName == 'BAIBLQ'){
								initForm(vm.$refs.wanConfigForm);
							}					
						},500)
					}
				})
			},
			specifyModifySubmit(){
				var vm = this, validFlag = true, params = {};
				params.id = vm.curSpecifyModifyRow.id;
				params.platform = vm.curSpecifyModifyRow.platform;
				params.serial_number = vm.curSpecifyModifyRow.serial_number;
				params.host_name = vm.curSpecifyModifyRow.host_name;

				vm.$refs.basicForm.validate(function(valid){
					if(valid){
						params.bands_support = vm.basicForm.bands_support;
						params.band_width = vm.basicForm.band_width;
						params.subframe_assignment = vm.basicForm.subframe_assignment;
						params.special_subframe_patterns = vm.basicForm.special_subframe_patterns;
						params.plmn_id = vm.basicForm.plmn_id;
						params.tac = vm.basicForm.tac;
						params.cell_identity = vm.basicForm.cell_identity;
						params.phycellid = vm.basicForm.phycellid;
						params.root_sequence_index = vm.basicForm.root_sequence_index;
						params.mme_ip = vm.basicForm.mmeGroup.toString();
						params.carrier_mode = vm.basicForm.carrier_mode;
						params.halob_enable = vm.basicForm.halob_enable;
						params.timeZoneUtc = vm.basicForm.timeZoneUtc;
						if(vm.platform == 'NBIOT'){
							params.ul_frequency = vm.basicForm.ul_frequency.substring(0,vm.basicForm.ul_frequency.indexOf("("));
						}
						if(vm.curProductName == "DXDF"){
							params.txPower = (vm.basicForm.dxdfTxPower||[]).join(",");
							params.frequency = vm.basicForm.frequency;
						}else{
							params.txPower = vm.basicForm.txPower;
							params.frequency = vm.basicForm.frequency.substring(0,vm.basicForm.frequency.indexOf("("));
						}
						//4860 cell2
						if((vm.curProductName == 'CR-B4860' || vm.curProductName == 'MLN') && vm.showCell2 == true){
							var cellFrequency = vm.basicForm.cellFrequency.substring(0,vm.basicForm.cellFrequency.indexOf("("));
							params.cell2params = JSON.stringify({
								plmn_id: vm.basicForm.cellPlmnId,
								band_width: vm.basicForm.cellBandwidth,
								frequency: cellFrequency,
								cell_identity: vm.basicForm.cellEci,
								phycellid: vm.basicForm.cellPhycellid,
								root_sequence_index: vm.basicForm.cellRootIndex,
								mme_ip: vm.basicForm.cellMmeGroup.toString()
							})
						}else{
							delete params.cell2params;
						}
					}else{
						validFlag = false
					}
				})
				if(vm.showSetting && vm.curProductName != 'DXDF'){
					params.ipsec_switch = vm.settingForm.ipsec_switch?1:0;
					params.ipsec_enable = vm.settingForm.ipsec_enable;
					params.ipsec_rightikeport = vm.settingForm.ipsec_rightikeport;
					params.left_interface = vm.settingForm.left_interface;
					params.ipsecList = JSON.stringify(vm.settingForm.ipsecList);
					vm.$refs.settingForm.validate(function(valid){
						if(valid){
						}else{
							validFlag = false
						}
					})
				}
				//wan config
				if(vm.curProductName == 'CR-B4860' || vm.curProductName == 'BAIBLQ'){
					params.wanSendEnable = vm.wanConfigForm.wanSendEnable;
					params.wanBindingList = JSON.stringify(vm.wanConfigForm.wanBindingList);
					vm.$refs.wanConfigForm.validate(function(valid){
						if(valid){
						}else{
							validFlag = false
						}
					})
				}
				if(validFlag){
					if(vm.checkParamChange()){//返回true为改变
						axios.post("${ctx}/SON/SelfConfiguration/updateSelfConfigPlanning.action",stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message:'<%=rb.getString("ChengGong")%>',
		    						type:'success',
		    					})
                                vm.$refs.ctablePlan.refresh();
                                vm.specifyModifyShow = false;	
							}else{
								vm.$message.error(data["message"])
							}
						})
					}else{
						vm.$message('<%=rb.getString("WuCanShuBianHua")%>');
					}
				}
			},
			specifyModifyCancel(){
				var vm = this;
				vm.specifyModifyShow = false;
				// 添加表单重置？？？？？？？
				if(vm.$refs.wanConfigForm) {
					vm.$refs.wanConfigForm.resetFields();
					vm.$refs.wanConfigForm.clearValidate();
				}
			},
			checkValueChange(oriArr,curArr){//返回true为没有改变，返回false为改变
				var isFlag=true;
				if(oriArr.length != curArr.length){
					isFlag = false;
				}else{
					for(let i=0;i<oriArr.length;i++){
						var oriArrItem = {};
						var curArrItem = {};
						Object.keys(oriArr[i]).sort().map(function(key){
							oriArrItem[key] = oriArr[i][key]
						})
						Object.keys(curArr[i]).sort().map(function(key){
							curArrItem[key] = curArr[i][key]
						})
						for(var oriKey in oriArrItem){
							for(var curKey in curArrItem){
								if(oriKey == curKey && oriArrItem[oriKey] != curArrItem[curKey]){
									isFlag =  false;
								}
							}
						}
					}
				}
				return isFlag
			},
			checkParamChange(){
				var vm = this, changeFlag = true;
				if(isFormChanged(vm.$refs.basicForm)){
					changeFlag = false;
				}
				if(vm.showSetting && vm.curProductName != 'DXDF'){
					if(isFormChanged(vm.$refs.settingForm)){
						changeFlag = false;
					}
					
					if(vm.checkValueChange(vm.settingForm.defaultIpsecList,vm.settingForm.ipsecList) == false){
						changeFlag = false;
					}
				}
				if(vm.curProductName == 'CR-B4860' || vm.curProductName == 'BAIBLQ'){
					if(isFormChanged(vm.$refs.wanConfigForm)){
						changeFlag = false;
					}
					if(vm.checkValueChange(vm.wanConfigForm.originalWanBindingList, vm.wanConfigForm.wanBindingList) == false){
						changeFlag = false;
					}
				}
				if(changeFlag){//没有变化
					return false;
				}else{
					return true;
				}
			},
			//删除 20240923:此处删除与 5g 和 cpe处理的逻辑不同（这俩网元则直接删除表数据），enb 网元，虽然删除调用了接口，实际只是删除了临时表数据，还需点击左下角提交时触发真实的表数据删除；
			specifyDeleteClick(row, evt){
				var vm = this,  params = { id: row.id, platform: row.platform };
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/delSelfConfigPlans.action",stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$refs.ctablePlan.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {})
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.coordinateTacImportShow = false;
        		vm.coordinateTacModifyShow = false;
				vm.enbIdTacModifyShow = false;
				vm.snAddShow = false;
			},
			handerClose(){
				this.$refs.menu_ipsec.hide();
			},
			addMME(flag){
				var vm = this, value = '', str = '', msg = '<%=rb.getString("IPDiZhi")%>';
				if(flag == 'cell'){
					value = vm.basicForm.cellMme;
					if(value == ''){
						vm.cellErrorMsg = msg;
					}else{
						if(isValidIP(value)){
							str = value;
							if(vm.basicForm.cellMmeGroup.indexOf(str) == -1){
								vm.basicForm.cellMmeGroup.push(str);
								vm.basicForm.cellMme = '';
								vm.cellErrorMsg = '';
							}else{
								vm.cellErrorMsg = '<%=rb.getString("YiCunZai")%>';
							}
						}else{
							vm.cellErrorMsg = msg;
						}
					}
				}else{
					value = vm.basicForm.mme;
					if(value == ''){
						vm.errorMsg = msg;
					}else{
						if(isValidIP(value)){
							str = value;
							if(vm.basicForm.mmeGroup.indexOf(str) == -1){
								vm.basicForm.mmeGroup.push(str);
								vm.basicForm.mme = '';
								vm.errorMsg = '';
							}else{
								vm.errorMsg = '<%=rb.getString("YiCunZai")%>';
							}
						}else{
							vm.errorMsg = msg;
						}
					}
				}
			},
			removeMME(item, flag){
				var vm = this, group = '', tip = '';
				if(flag == 'cell'){
					group = vm.basicForm.cellMmeGroup;
					tip = vm.cellErrorMsg;
				}else{
					group = vm.basicForm.mmeGroup;
					tip = vm.errorMsg;
				}
				var index = group.indexOf(item);
				if(index !== -1){
					group.splice(index,1)
				}
				tip = '';
			},
			optClickIpsec(row,ev){
				var vm = this, showEditOrDelete = false;
				vm.rowDataIpsec = row;
				if(vm.curType == "view"){
					showEditOrDelete = false;
				}else {
					showEditOrDelete = true;
				}
				vm.menus_ipsec = [
					{label:"<%=rb.getString("ChaKan")%>",cls:"el-icon el-icon-operation-info",code:"info"},
					{label:"<%=rb.getString("XiuGai")%>",cls:"el-icon el-icon-operation-edit",code:"edit", show: showEditOrDelete},
					{label:"<%=rb.getString("ShanChu")%>",cls:"el-icon el-icon-operation-delete",code:"del", show: showEditOrDelete},
				]
				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.menu_ipsec.show(ev);
				})
			},
			clickMenuIpsec(ev){
				var codes = {
					info: this.viewIpsec,
					edit: this.editIpsec,
					del: this.delIpsec
				}
				if(codes[ev.code]){
					codes[ev.code]();
				}
			}, 		
			//add ipsec tunnel
			addIpsec(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEdit.action',
				vm.slideTitle = '<%=rb.getString("TianJia")%> Ipsec Tunnel';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '700px';
				vm.slideWidth = '820px';
				vm.operTypeIpsec = 'add';
				vm.itemType = 'ipsec';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false
	    	    });
			},		
			editIpsec(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEdit.action',
				vm.slideTitle = '<%=rb.getString("XiuGai")%> Ipsec Tunnel';
				vm.slideFooter = true;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '700px';
				vm.slideWidth = '820px';
				vm.operTypeIpsec = 'edit';
				vm.itemType = 'ipsec';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-ipsec");
	    	    });
			},
			viewIpsec(){
				var vm = this;
				vm.slideUrl = '${ctx}/SON/SelfConfiguration/goSelfParamConfigIpsecEdit.action',
				vm.slideTitle = '<%=rb.getString("ChaKan")%> Ipsec Tunnel';
				vm.slideFooter = false;
				vm.slideHeader = true;
				vm.slidePosition = 'left';
				vm.slideHeight = '700px';
				vm.slideWidth = '820px';
				vm.operTypeIpsec = 'view';
				vm.itemType = 'ipsec';
				vm.$refs.slide.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    	eventBus.$emit("edit-ipsec");
	    	    });
			},
			saveSlide(){
				if(this.itemType == 'ipsec'){
					eventBus.$emit("save-ipsec");
				}
			},
			cancelSlide(){
				this.$refs.slide.hide();
			},
			delIpsec(){
				var vm = this;
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					var ipsecArr = vm.settingForm.ipsecList.map(function(item){
						return item.IPSEC_INDEX;
					})
					var index = ipsecArr.indexOf(vm.rowDataIpsec.IPSEC_INDEX);
					vm.settingForm.ipsecList.splice(index,1);
					var length = vm.settingForm.ipsecList.length;
					for(let i=0;i<length;i++){
						vm.settingForm.ipsecList[i].IPSEC_INDEX = i+1;
					}
				}).catch(() => {
					
				})
			},
			changeToEarfcn(type){
				if(type == 'dl'){
					var value = this.basicForm.frequency;
					var showFlag = value.indexOf("(");
					if(showFlag == -1);
					else this.basicForm.frequency = this.basicForm.frequency.substring(0,showFlag);
				}else if(type == 'ul'){
					var value = this.basicForm.ul_frequency;
					var showFlag = value.indexOf("(");
					if(showFlag == -1);
					else this.basicForm.ul_frequency = this.basicForm.ul_frequency.substring(0,showFlag);
				}
			},
			// 频率转换
            changeToFrequency(type) { 
				if(type == 'dl'){
					var value = this.basicForm.frequency;
					if(value == "");
					else this.basicForm.frequency =  translateToFre(this.basicForm.frequency);
				}else if(type == 'ul'){
					var value = this.basicForm.ul_frequency;
					if(value == "");
					else this.basicForm.ul_frequency =  translateToFre(this.basicForm.ul_frequency);
				}
            },
			cellChangeToFrequency(type) { 
				if(type == 'dl'){
					var value = this.basicForm.cellFrequency;
					if(value == "");
					else this.basicForm.cellFrequency =  translateToFre(this.basicForm.cellFrequency);
				}
            },
			cellChangeToEarfcn(type){ //cellFrequency
				if(type == 'dl'){
					var value = this.basicForm.cellFrequency;
					var showFlag = value.indexOf("(");
					if(showFlag == -1);
					else this.basicForm.cellFrequency = this.basicForm.cellFrequency.substring(0,showFlag);
				}
			},
            changeHalob(val){
				var vm = this;
				if(val == "1"){
					vm.showSetting = false;
					vm.showMME = false;
				}else{
					vm.showMME = true;
					if(vm.platform == 'NBIOT'){
						vm.showSetting = false;
					}else{
						vm.showSetting = true;
						setTimeout(function(){
							initForm(vm.$refs.settingForm);
						},10)
					}
				}
			},
			tunnelEnableFmt(row,column,value,index){
				return value == "1" ? "enable" : "disable";
			},
			//----------------------ipsec information
			//详情
			specifyInfoClick(row, evt){
				var vm = this;
				vm.specifyInfoShow = true;
				vm.specifyModifyShow = false;
				vm.specifyImportShow = false;
				vm.coordinateTacImportShow = false;
        		vm.coordinateTacModifyShow = false;
				vm.enbIdTacModifyShow = false;
				vm.snAddShow = false;
				vm.platform = row.platform;
				vm.commonParamsMethod();
				if(vm.platform == "intel"){
					vm.ipsecInfoShowMode = true;
					vm.ipsecInfoShowItem = true;
					vm.ipsecInfoShownbi = true;
					vm.ipsecInfoShowBasic = true;
					vm.ipsecInfoMapFlag = false;
				}else if(vm.platform == "qa"){
					vm.ipsecInfoMapFlag = true;
					vm.ipsecInfoShownbi = true;
					vm.ipsecInfoShowBasic = true;
					vm.ipsecInfoShowMode = false;
					vm.ipsecInfoShowItem = false;
				}else if(vm.platform == 'NBIOT'){
					vm.ipsecInfoShowMME = true;
					vm.ipsecInfoShowMode = false;
					vm.ipsecInfoShowItem = false;
					vm.ipsecInfoShownbi = false;
					vm.ipsecInfoShowBasic = false;
					vm.ipsecInfoMapFlag = false;
					vm.frequencyLabel = 'DL Frequency'
				}
				var params = { id : row.id };
				axios.post("${ctx}/SON/SelfConfiguration/getSelfConfigPlanningInfos.action",stringify(params)).then(function(response){
					var data = response.data;
					vm.paramList = data.paramList;
					//如果SAS开关打开，则该参数不可配置
			    	if(SASEnble == "1"){
			    		vm.ipsecInfoSasDisabled = true;
			    	} 
			    	if(data == "" || data == null){
					}else{
						//接口有数据时
						vm.specifySN = data.serial_number;
						vm.cellName = data.host_name;
						//basic 
						vm.bands_support = data.bands_support;						
						if(vm.curProductName == "DXDF"){
							if(data.band_width != null){
								vm.band_width = data.band_width;
							}

							if(data.frequency == '' || data.frequency == null){
								vm.frequency = ''
							}else{
								vm.frequency = data.frequency;
							}
						}else{
							if(data.band_width != null){
								if(data.band_width == 'n25'){
									vm.band_width = '5MHz';
								}else if(data.band_width == 'n50'){
									vm.band_width = '10MHz';
								}else if(data.band_width == 'n75'){
									vm.band_width = '15MHz';
								}else{
									vm.band_width = '20MHz';
								}
							}
							if(data.frequency != null){
								if(translateToFre(data.frequency) == false){
								}else{
									vm.frequency = translateToFre(data.frequency)
								}
							}
						}
						if(vm.platform == 'NBIOT' && data.ul_frequency != null){
							if(translateToFre(data.ul_frequency) == false){
								
							}else{
								vm.ul_frequency = translateToFre(data.ul_frequency)
							}
						}
						if(data.subframe_assignment != null){
							if(data.subframe_assignment == '0'){
								vm.subframe_assignment = '0(DL:UL = 1:3)';
							}else if(data.subframe_assignment == '1'){
								vm.subframe_assignment = '1(DL:UL = 2:2)';
							}else if(data.subframe_assignment == '2'){
								vm.subframe_assignment = '2(DL:UL = 3:1)';
							}else{
								//data.subframe_assignment == '6'
								vm.subframe_assignment = '6(DL:UL = 3:5)';
							}
						}
						if(data.special_subframe_patterns != null){
							vm.special_subframe_patterns = data.special_subframe_patterns;							
						}
						vm.plmn_id = data.plmn_id;
						vm.tac = data.tac;
						vm.cell_identity = data.cell_identity;
						vm.phycellid = data.phycellid;
						vm.root_sequence_index = data.root_sequence_index;			
						vm.mmeStr = data.mme_ip;
						vm.carrier_mode = data.carrier_mode;
						if(data.halob_enable != null){
							if(data.halob_enable == '0'){
								vm.halob_enable = '<%=rb.getString("HalobGuanBi")%>';
							}else{
								//data.halob_enable == '1'
								vm.halob_enable = '<%=rb.getString("HalobKaiQi")%>';
							}
						}
						if(data.timeZoneUtc == '' || data.timeZoneUtc == null){
							vm.timeZoneUtc = ''
						}else{
							vm.timeZoneList.map((item,index)=>{
								if(item.value == data.timeZoneUtc){
									vm.timeZoneUtc = item.label;
								}
							})
						}
						if(data.txPower == '' || data.txPower == null){
							vm.txPower = ''
						}else{
							if(vm.curProductName == "DXDF"){
								var power = data.txPower.split(",");
								//将每个值都拼接 ‘dBm’
								vm.txPower = power.map((item,index)=>{
									return item + 'dBm';
								}).join(",");
							}else{
								vm.powerList.map((item,index)=>{
									if(item.value == data.txPower){
										vm.txPower = item.label;
									}
								})
							}
						}
						//4860 判断 data 中 是否含有 cell2params
						if(data.cell2params && (vm.curProductName == 'CR-B4860' || vm.curProductName == 'MLN')){
							vm.showInfoCell2 = true;
							vm.cellPlmnId = data.cell2params.plmn_id;
							vm.cellEci = data.cell2params.cell_identity; // eci
							vm.cellPhycellid = data.cell2params.phycellid; //pci
							vm.cellRootIndex = data.cell2params.root_sequence_index;
							vm.cellInfoMme = data.cell2params.mme_ip;							
							var bandWith = data.cell2params.band_width,
								frequencyInfo = data.cell2params.frequency;
							if(bandWith != null){
								if(bandWith == 'n25'){
									vm.cellBandwidth = '5MHz';
								}else if(bandWith == 'n50'){
									vm.cellBandwidth = '10MHz';
								}else if(bandWith == 'n75'){
									vm.cellBandwidth = '15MHz';
								}else{
									vm.cellBandwidth = '20MHz';
								}
							}
							if(frequencyInfo != null){
								if(translateToFre(frequencyInfo) == false){
								}else{
									vm.cellFrequency = translateToFre(frequencyInfo)
								}
							}
						}
						//settings
						vm.ipsec_switch = data["ipsec_switch"]==0?false:true;
						vm.ipsec_enable = data.ipsec_enable;
						vm.ipsec_rightikeport = data.ipsec_rightikeport;
						vm.left_interface = data.left_interface;
						vm.ipsecList = data.ipsecList;
						if(data.halob_enable == "1"){//打开
							vm.ipsecInfoShowMME = false;
							vm.ipsecInfoShowSetting = false;
						}else{//关闭
							vm.ipsecInfoShowMME = true;
							
							if(vm.platform == 'NBIOT'){
								vm.ipsecInfoShowSetting = false;
							}else{
								vm.ipsecInfoShowSetting = true;
							}
						}
						if(vm.platform == "qa"){
							if(data.sync_type == "PTP"){
								vm.ipsecInfoShowPtp = true;
							}else{
								vm.ipsecInfoShowPtp = false;
							}
						}
						if(vm.mode_switch == 'unicast'){
							vm.ipsecInfoShowAddress = true;
						}else{
							vm.ipsecInfoShowAddress = false;
						}
						if(vm.curProductName == 'CR-B4860' || vm.curProductName == 'BAIBLQ'){
							vm.infoWanSendEnable = data.wanSendEnable;
							vm.infoWanBindingList = (data.wanBindingList && data.wanBindingList.length > 0) ? data.wanBindingList : [];
						}
					}
				})
			},
			specifyInfoCancel(){
				var vm = this;
				vm.specifyInfoShow = false;
			},
			//公共参数映射
			commonParamsMethod(){
				var vm = this,
					timeStr = "Africa/Abidjan,Africa/Accra,Africa/Addis_Ababa,Africa/Algiers,Africa/Asmara,Africa/Bamako,Africa/Bangui,Africa/Banjul,Africa/Bissau,Africa/Blantyre,Africa/Brazzaville,Africa/Bujumbura,Africa/Cairo,Africa/Casablanca,Africa/Ceuta,Africa/Conakry,Africa/Dakar,Africa/Dar_es_Salaam,Africa/Djibouti,Africa/Douala,Africa/El_Aaiun,Africa/Freetown,Africa/Gaborone,Africa/Harare,Africa/Johannesburg,Africa/Juba,Africa/Kampala,Africa/Khartoum,Africa/Kigali,Africa/Kinshasa,Africa/Lagos,Africa/Libreville,Africa/Lome,Africa/Luanda,Africa/Lubumbashi,Africa/Lusaka,Africa/Malabo,Africa/Maputo,Africa/Maseru,Africa/Mbabane,Africa/Mogadishu,Africa/Monrovia,Africa/Nairobi,Africa/Ndjamena,Africa/Niamey,Africa/Nouakchott,Africa/Ouagadougou,Africa/Porto-Novo,Africa/Sao_Tome,Africa/Tripoli,Africa/Tunis,Africa/Windhoek,America/Adak,America/Anchorage,America/Anguilla,America/Antigua,America/Araguaina,America/Argentina/Buenos_Aires,America/Argentina/Catamarca,America/Argentina/Cordoba,America/Argentina/Jujuy,America/Argentina/La_Rioja,America/Argentina/Mendoza,America/Argentina/Rio_Gallegos,America/Argentina/Salta,America/Argentina/San_Juan,America/Argentina/San_Luis,America/Argentina/Tucuman,America/Argentina/Ushuaia,America/Aruba,America/Asuncion,America/Atikokan,America/Bahia,America/Bahia_Banderas,America/Barbados,America/Belem,America/Belize,America/Blanc-Sablon,America/Boa_Vista,America/Bogota,America/Boise,America/Cambridge_Bay,America/Campo_Grande,America/Cancun,America/Caracas,America/Cayenne,America/Cayman,America/Chicago,America/Chihuahua,America/Costa_Rica,America/Creston,America/Cuiaba,America/Curacao,America/Danmarkshavn,America/Dawson,America/Dawson_Creek,America/Denver,America/Detroit,America/Dominica,America/Edmonton,America/Eirunepe,America/El_Salvador,America/Fortaleza,America/Glace_Bay,America/Godthab,America/Goose_Bay,America/Grand_Turk,America/Grenada,America/Guadeloupe,America/Guatemala,America/Guayaquil,America/Guyana,America/Halifax,America/Havana,America/Hermosillo,America/Indiana/Indianapolis,America/Indiana/Knox,America/Indiana/Marengo,America/Indiana/Petersburg,America/Indiana/Tell_City,America/Indiana/Vevay,America/Indiana/Vincennes,America/Indiana/Winamac,America/Inuvik,America/Iqaluit,America/Jamaica,America/Juneau,America/Kentucky/Louisville,America/Kentucky/Monticello,America/Kralendijk,America/La_Paz,America/Lima,America/Los_Angeles,America/Lower_Princes,America/Maceio,America/Managua,America/Manaus,America/Marigot,America/Martinique,America/Matamoros,America/Mazatlan,America/Menominee,America/Merida,America/Metlakatla,America/Mexico_City,America/Miquelon,America/Moncton,America/Monterrey,America/Montevideo,America/Montserrat,America/Nassau,America/New_York,America/Nipigon,America/Nome,America/Noronha,America/North_Dakota/Beulah,America/North_Dakota/Center,America/North_Dakota/New_Salem,America/Ojinaga,America/Panama,America/Pangnirtung,America/Paramaribo,America/Phoenix,America/Port_of_Spain,America/Port-au-Prince,America/Porto_Velho,America/Puerto_Rico,America/Rainy_River,America/Rankin_Inlet,America/Recife,America/Regina,America/Resolute,America/Rio_Branco,America/Santa_Isabel,America/Santarem,America/Santiago,America/Santo_Domingo,America/Sao_Paulo,America/Scoresbysund,America/Sitka,America/St_Barthelemy,America/St_Johns,America/St_Kitts,America/St_Lucia,America/St_Thomas,America/St_Vincent,America/Swift_Current,America/Tegucigalpa,America/Thule,America/Thunder_Bay,America/Tijuana,America/Toronto,America/Tortola,America/Vancouver,America/Whitehorse,America/Winnipeg,America/Yakutat,America/Yellowknife,Antarctica/Casey,Antarctica/Davis,Antarctica/DumontDUrville,Antarctica/Macquarie,Antarctica/Mawson,Antarctica/McMurdo,Antarctica/Palmer,Antarctica/Rothera,Antarctica/Syowa,Antarctica/Troll,Antarctica/Vostok,Arctic/Longyearbyen,Asia/Aden,Asia/Almaty,Asia/Amman,Asia/Anadyr,Asia/Aqtau,Asia/Aqtobe,Asia/Ashgabat,Asia/Baghdad,Asia/Bahrain,Asia/Baku,Asia/Bangkok,Asia/Beirut,Asia/Bishkek,Asia/Brunei,Asia/Chita,Asia/Choibalsan,Asia/Colombo,Asia/Damascus,Asia/Dhaka,Asia/Dili,Asia/Dubai,Asia/Dushanbe,Asia/Gaza,Asia/Hebron,Asia/Ho_Chi_Minh,Asia/Hong_Kong,Asia/Hovd,Asia/Irkutsk,Asia/Jakarta,Asia/Jayapura,Asia/Jerusalem,Asia/Kabul,Asia/Kamchatka,Asia/Karachi,Asia/Kathmandu,Asia/Khandyga,Asia/Kolkata,Asia/Krasnoyarsk,Asia/Kuala_Lumpur,Asia/Kuching,Asia/Kuwait,Asia/Macau,Asia/Magadan,Asia/Makassar,Asia/Manila,Asia/Muscat,Asia/Nicosia,Asia/Novokuznetsk,Asia/Novosibirsk,Asia/Omsk,Asia/Oral,Asia/Phnom_Penh,Asia/Pontianak,Asia/Pyongyang,Asia/Qatar,Asia/Qyzylorda,Asia/Rangoon,Asia/Riyadh,Asia/Sakhalin,Asia/Samarkand,Asia/Seoul,Asia/Shanghai,Asia/Singapore,Asia/Srednekolymsk,Asia/Taipei,Asia/Tashkent,Asia/Tbilisi,Asia/Thimphu,Asia/Tokyo,Asia/Ulaanbaatar,Asia/Urumqi,Asia/Ust-Nera,Asia/Vientiane,Asia/Vladivostok,Asia/Yakutsk,Asia/Yekaterinburg,Asia/Yerevan,Atlantic/Azores,Atlantic/Bermuda,Atlantic/Canary,Atlantic/Cape_Verde,Atlantic/Faroe,Atlantic/Madeira,Atlantic/Reykjavik,Atlantic/South_Georgia,Atlantic/St_Helena,Atlantic/Stanley,Australia/Adelaide,Australia/Brisbane,Australia/Broken_Hill,Australia/Currie,Australia/Darwin,Australia/Eucla,Australia/Hobart,Australia/Lindeman,Australia/Lord_Howe,Australia/Melbourne,Australia/Perth,Australia/Sydney,Europe/Amsterdam,Europe/Andorra,Europe/Athens,Europe/Belgrade,Europe/Berlin,Europe/Bratislava,Europe/Brussels,Europe/Bucharest,Europe/Budapest,Europe/Busingen,Europe/Chisinau,Europe/Copenhagen,Europe/Dublin,Europe/Gibraltar,Europe/Guernsey,Europe/Helsinki,Europe/Isle_of_Man,Europe/Istanbul,Europe/Jersey,Europe/Kaliningrad,Europe/Kiev,Europe/Lisbon,Europe/Ljubljana,Europe/London,Europe/Luxembourg,Europe/Madrid,Europe/Malta,Europe/Mariehamn,Europe/Minsk,Europe/Monaco,Europe/Moscow,Europe/Oslo,Europe/Paris,Europe/Podgorica,Europe/Prague,Europe/Riga,Europe/Rome,Europe/Samara,Europe/San_Marino,Europe/Sarajevo,Europe/Simferopol,Europe/Skopje,Europe/Sofia,Europe/Stockholm,Europe/Tallinn,Europe/Tirane,Europe/Uzhgorod,Europe/Vaduz,Europe/Vatican,Europe/Vienna,Europe/Vilnius,Europe/Volgograd,Europe/Warsaw,Europe/Zagreb,Europe/Zaporozhye,Europe/Zurich,Indian/Antananarivo,Indian/Chagos,Indian/Christmas,Indian/Cocos,Indian/Comoro,Indian/Kerguelen,Indian/Mahe,Indian/Maldives,Indian/Mauritius,Indian/Mayotte,Indian/Reunion,Pacific/Apia,Pacific/Auckland,Pacific/Bougainville,Pacific/Chatham,Pacific/Chuuk,Pacific/Easter,Pacific/Efate,Pacific/Enderbury,Pacific/Fakaofo,Pacific/Fiji,Pacific/Funafuti,Pacific/Galapagos,Pacific/Gambier,Pacific/Guadalcanal,Pacific/Guam,Pacific/Honolulu,Pacific/Johnston,Pacific/Kiritimati,Pacific/Kosrae,Pacific/Kwajalein,Pacific/Majuro,Pacific/Marquesas,Pacific/Midway,Pacific/Nauru,Pacific/Niue,Pacific/Norfolk,Pacific/Noumea,Pacific/Pago_Pago,Pacific/Palau,Pacific/Pitcairn,Pacific/Pohnpei,Pacific/Port_Moresby,Pacific/Rarotonga,Pacific/Saipan,Pacific/Tahiti,Pacific/Tarawa,Pacific/Tongatapu,Pacific/Wake,Pacific/Wallis",
					timeList = timeStr.split(","),
					powerList = ['-20dBm(0.01mW)','-10dBm(0.1mW)','0dBm(1.0mW)','1dBm(1.3mW)','2dBm(1.6mW)','3dBm(2.0mW)','4dBm(2.5mW)','5dBm(3.2mW)','6dBm(4.0mW)','7dBm(5.0mW)','8dBm(6.3mW)','9dBm(7.9mW)','10dBm(10mW)','11dBm(13mW)','12dBm(16mW)','13dBm(20mW)','14dBm(25mW)','15dBm(32mW)','16dBm(40mW)','17dBm(50mW)','18dBm(63mW)','19dBm(79mW)','20dBm(100mW)','21dBm(126mW)','22dBm(158mW)','23dBm(200mW)','24dBm(251mW)','25dBm(316mW)','26dBm(398mW)','27dBm(501mW)'],
					powerValList = [-20,-10,0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27];
				
				vm.timeZoneList = timeList.map(item=>{
					return{label:item,value:item}
				})
				vm.powerList = powerList.map((item,index)=>{
					return {label:item,value:powerValList[index]}
				})
			},
			//搜索
			queryPlan(){
				Object.assign(this.params_plan,this.params_plan_form);
			},
			bandWidthFmt(row,column,value,index){
				var vm = this;
				
				if(vm.curProductName == "DXDF"){
					if(value == "6"){
						return "6";
					}else if(value == "15"){
						return "15"
					}else if(value == "25"){
						return "25"
					}else if(value == "50"){
						return "50"
					}else if(value == "75"){
						return "75"
					}else if(value == "100"){
						return "100"
					}
				}else{
					if(value == "n25"){
						return "5MHz";
					}else if(value == "n50"){
						return "10MHz"
					}else if(value == "n75"){
						return "15MHz"
					}else if(value == "n100"){
						return "20MHz"
					}
				}
			},
			bandwidthFmtOne(row,column,value,index) {
				var vm = this,
					rels = {
						'n25': '5MHz',
						'n50': '10MHz',
						'n75': '15MHz',
						'n100': '20MHz'
					};
				return rels[value]||'';
			},
			earfcnFmt(row,column,value,index){
				var vm = this;
				if(vm.curProductName == "DXDF"){
					if(value == "" || value == null){
						return "";
					}else{
						return value;
					}
				}else{
					if(value == "" || value == null){
						return "";
					}else{
						return translateToFre(value);
					}
				}
			},
			sfassignmentFmt(row,column,value,index){
				if(value == "1"){
					return "1(DL:UL = 2:2)"
				}else if(value =="2"){
					return "2(DL:UL = 3:1)"
				}else if(value == "0"){
					return "0(DL:UL = 1:3)"
				}else if(value == "6"){
					return "6(DL:UL = 3:5)"
				}else{
					return value
				}
			},
			//-----------------------------------TAC	
			//coordinate tac, enb id tac 切换
			tacConfigChange(val){
				var vm = this;
				vm.coordinateTacImportShow = false;
				vm.coordinateTacModifyShow = false;
				vm.enbIdTacModifyShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.snAddShow = false;
			},
			//coordinate 导入
			coordinateImportClick(){
				var vm = this;
				vm.coordinateTacImportShow = true;
				vm.coordinateTacModifyShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.enbIdTacModifyShow = false;
				vm.snAddShow = false;
				vm.$refs.mapForm.resetFields();
			},
			selectPolFile(){
				this.$refs['file_polygon'].click();
			},
			beforeUploadPolygon(file){
				var fd = new FormData(), vm = this,
				config = {
					headers: { 'Content-Type': 'multipart/form-data' }
				};
				fd.append('uploadFile',file);
				 //文件流
				axios.post("${ctx}/SON/SelfConfiguration/importSelfCoordinateTac.action",fd,config).then(function(response){
					var data = response.data
					if(data["success"]){
			     		vm.$refs.ctableMap.refresh();
			     		vm.coordinateImportCancel();
					}else{
						vm.$message.error(data["message"])
					}
					vm.mapLoading = false;
					vm.mapSubmitDisabled = false;
				})
			},
			fileChangePolygon(file,fileList){
				var vm = this;
				vm.mapForm.file_polygon = file.name;
			},
			//coordinate 导入提交
			coordinateImportSubmit(){
				var vm = this;
				vm.$refs.mapForm.validate((valid) => {
					if(valid){
						vm.$refs.upload_polygon.submit();
						vm.mapLoading = true; // 页面loading
						vm.mapSubmitDisabled = true; //提交按钮置灰
					}
				})
			},
			//导入取消
			coordinateImportCancel(){
				var vm = this;
				vm.coordinateTacImportShow = false;
				vm.mapForm.file_polygon = '';
				vm.$refs.upload_polygon.clearFiles();
				vm.$refs.mapForm.resetFields();
			},
			//coordinate 清除
			coordinateClearClick(){
				var vm = this;
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/clearSelfCoordinateTac.action").then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					})
	    					vm.$refs.ctableMap.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
				vm.coordinateModifyCancel();
				vm.coordinateTacImportShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
			},
			//coordinate 修改
			coordinateModifyClick(rows,evt){
				var vm = this;
				vm.rowDataMap = rows;
				vm.coordinateTacModifyShow = true;
				vm.coordinateTacImportShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.modifyMapForm.groupName = rows.groupName;
				vm.modifyMapForm.tac = rows.tac;
				vm.idRangeStart = (rows.eciRange).split("-")[0];
				vm.idRangeEnd = (rows.eciRange).split("-")[1];
			},			
			//coordinate 修改提交
			coordinateModifySubmit(){    
				var vm = this;
				vm.$refs.modifyMapForm.validate(function(valid){
					if(valid){
						var params = {
							id: vm.rowDataMap.id,
							tac: vm.modifyMapForm.tac,
							groupName: vm.modifyMapForm.groupName,
							eciRange: vm.modifyMapForm.idRange
						}
						axios.post("${ctx}/SON/SelfConfiguration/updateSelfCoordinateTacById.action",stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message:'<%=rb.getString("ChengGong")%>',
		    						type:'success'
		    					})
		    					vm.$refs.ctableMap.refresh();
								vm.coordinateTacModifyShow = false;
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				})
			},
			//coordinate 修改取消
			coordinateModifyCancel(){
				var vm = this;
				vm.coordinateTacModifyShow = false;
			},
			//coordinate 删除
			coordinateDeleteClick(row,evt){
				var vm = this, params = { id: row.id };
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/delSelfCoordinateTacById.action",stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					})
	    					vm.$refs.ctableMap.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
				vm.coordinateModifyCancel();
				vm.coordinateTacImportShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
			},
			//sn
			snAddClick(){
				var vm = this;
				vm.snAddShow = true;
				vm.coordinateTacImportShow = false;
				vm.coordinateTacModifyShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.enbIdTacModifyShow = false;
				vm.operSnType = 'add';
				vm.isExecute = '0';
				vm.snNameDisabled = false;
				vm.addSnForm.mmeGroup = [];
				vm.AddModifySnTitle = '<%=rb.getString("TianJia")%>';
				vm.$refs.addSnForm.resetFields();
			},
			//sn list view
			snListClick(row,evt){
				var vm = this;
				vm.snListShow = true;
				vm.snSelections = [];
				vm.snParams.policyId = row.policyId;
				vm.snParams.groupName = row.groupName;
			},
			queryDetect(val){
				var vm = this;
				vm.snParams.searchText = val;	
			},
			selectChange(selection) {
				this.snSelections = selection;
			},
			delBatchSn() {
				var vm = this,
					list = (vm.snSelections||[]).map(function(item){
						return item.id;
					}),
					params = {
						id: list.join(',')
					};
				vm.snAddShow = false;
				//将新建，修改弹窗关闭 
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post('${ctx}/SON/SelfConfiguration/delSelfSNTacById.action',stringify(params)).then(function(res){
						var data = res.data;
						if(data['success']) {
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type: 'success'
							});
							vm.$refs.snCtableMap.refresh();
							vm.$refs.snTableList.refresh();
							vm.closeSnDialog();
						}else {
							vm.$message.error(data["message"])
						}
					}).catch(function(){});
				}).catch()
			},
			closeSnDialog(){
				var vm = this;
				vm.isExecute = '0';
				vm.snSelections = [];
				vm.$refs.snTableList.clearSelection(); //清空已选数据
				vm.snListShow = false;
			},
			addMmeIp(){
				var vm = this, value = vm.addSnForm.mme, str = '', msg = "<%=rb.getString("IPDiZhi")%>"
				vm.$refs.addSnForm.clearValidate('mmeIp');
				if(value == ''){
					vm.errorMmeIpMsg = msg;
				}else{
					if(isValidIP(value)){
						str = value;
						if(vm.addSnForm.mmeGroup.indexOf(str) == -1){
							vm.addSnForm.mmeGroup.push(str);
							vm.addSnForm.mme = '';
							vm.errorMmeIpMsg = '';
						}else{
							vm.errorMmeIpMsg = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.errorMmeIpMsg = msg;
					}
				}
			},
			removeMmeIp(item){
				var vm = this, index = vm.addSnForm.mmeGroup.indexOf(item);
				if(index !== -1){
					vm.addSnForm.mmeGroup.splice(index,1)
				}
				vm.errorMmeIpMsg = '';
			},
			snAddSubmit(){
				var vm = this;
				//修改时校验参数是否发生变化
				vm.$refs.addSnForm.validate(function(valid){
					if(valid){
						vm.snSaveBtnDisabled = true;
						var params = {
							groupName: vm.addSnForm.groupName,
							tac: vm.addSnForm.tac,
							eci: vm.addSnForm.eci,
							serialNumber: vm.addSnForm.serialNumber,
							mmeIp: vm.addSnForm.mmeIp
						}
						if(vm.operSnType == 'update'){
							params.policyId = vm.snRowData.policyId;
						}
						axios.post("${ctx}/SON/SelfConfiguration/addSelfConfigAreaTacPolicy.action",stringify(params)).then(function(response){
							var data = response.data;
							vm.snSaveBtnDisabled = false;
							if(data["success"]){
								vm.$message({
		    						message:'<%=rb.getString("ChengGong")%>',
		    						type:'success'
		    					})
		    					vm.$refs.snCtableMap.refresh();
								vm.snAddCancel();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				})
			},
			snModifyClick(row,evt){
				var vm = this;
				vm.snRowData = row;
				vm.operSnType = 'update';
				vm.AddModifySnTitle = '<%=rb.getString("XiuGai")%>';
				vm.snNameDisabled = true;
				vm.addSnForm.groupName = row.groupName;
				vm.addSnForm.tac = row.tac;
				vm.addSnForm.eci = row.eciRange;
				if(row.mmeIp === '' || row.mmeIp === null || row.mmeIp === undefined){
					vm.addSnForm.mmeIp = '';
					vm.addSnForm.mmeGroup = [];
				}else{
					vm.addSnForm.mmeIp = row.mmeIp;
					vm.addSnForm.mmeGroup = row.mmeIp.split(",");
				}
				//sn 通过接口返回
				var params = { groupName: row.groupName, policyId : row.policyId, };
				axios.post("${ctx}/SON/SelfConfiguration/getAreaSNByGroupName.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data){
						var lastIdx = data.lastIndexOf(',');
						vm.addSnForm.serialNumber = data.substring(0,lastIdx);
					}
				})
				vm.snAddShow = true;
				vm.coordinateTacImportShow = false;
				vm.coordinateTacModifyShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.enbIdTacModifyShow = false;
			},
			snDeleteClick(row,evt){
				var vm = this, params = { id: row.id, groupName: row.groupName };
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/delSelfAreaTacById.action",stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					})
	    					vm.$refs.snCtableMap.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
				vm.snAddCancel();	
				vm.coordinateTacImportShow = false;
				vm.coordinateTacModifyShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.enbIdTacModifyShow = false;
			},
			//eNB ID - TAC add cancel
			snAddCancel(){
				var vm = this;
				vm.snAddShow = false;
				vm.addSnForm.mmeGroup = [];
				vm.$refs.addSnForm.resetFields();
			},
			//eNB ID - TAC add
			enbIdTacAddClick(){
				var vm = this;
				vm.enbIdTacModifyShow = true;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.operTacType = 'add';
				vm.enbIdTacDisabled = false;
				vm.snAddShow = false;
				vm.AddModifyTacTitle = '<%=rb.getString("TianJia")%>';
				vm.$refs.addTacForm.resetFields();
			},
			//eNB ID - TAC add submit
			enbIdTacAddSubmit(){
				var vm = this;
				vm.$refs.addTacForm.validate(function(valid){
					if(valid){
						var params = {
								tac : vm.addTacForm.tac,
								eciRange : vm.addTacForm.eciRange
						}
						if(vm.operTacType == 'add'){
							params.operType = 'add';
						}else{
							params.operType = 'update';
						}
						axios.post("${ctx}/SON/SelfConfiguration/setSelfAutoTac.action",stringify(params)).then(function(response){
							var data = response.data;
							if(data["success"]){
								vm.$message({
		    						message:'<%=rb.getString("ChengGong")%>',
		    						type:'success'
		    					})
		    					vm.$refs.enbIdTactable.refresh();
								vm.enbIdTacAddCancel();
							}else{
								vm.$message.error(data["message"])
							}
						})
					}
				})
			},
			//eNB ID - TAC add cancel
			enbIdTacAddCancel(){
				var vm = this;
				vm.enbIdTacModifyShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.snAddShow = false;
				vm.$refs.addTacForm.resetFields();
			},
			//eNB ID - TAC modify
			enbIdModifyClick(row,evt){
				var vm = this;
				vm.enbIdTacRowData = row
				vm.enbIdTacModifyShow = true;
				vm.snAddShow = false;
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.operTacType = 'update';
				vm.AddModifyTacTitle = '<%=rb.getString("XiuGai")%>';
				vm.enbIdTacDisabled = true;
				vm.addTacForm.tac = row.tac;
				vm.addTacForm.eciRange = row.eciRange;
			},
			//eNB ID - TAC delete
			enbIdDeleteClick(row,evt){
				var vm = this, params = { id: row.id };
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post("${ctx}/SON/SelfConfiguration/delSelfAutoTac.action",stringify(params)).then(function(response){
						var data = response.data;
						if(data["success"]){
							vm.$message({
	    						message:'<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					})
	    					vm.$refs.enbIdTactable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
				vm.enbIdTacModifyShow = false;
				//batch import
				vm.specifyModifyShow = false;
				vm.specifyInfoShow = false;
				vm.specifyImportShow = false;
				vm.snAddShow = false;
			},
			queryEnbLicense(text) {
				this.enbLicenseQuery.searchText = text;
				this.enbLicenseQuery.productType = this.addOrEditForm.productType;
			},
        	//字段格式化
        	executeStatus(row,column,value,rowIndex){
				if ('0' == value) {
					return "<%=rb.getString("WeiZhiXing")%>";
				} else if ('1' == value){
					return "<%=rb.getString("ZhengZaiZhiXing")%>";
				} else if ('2' == value){
					return "<%=rb.getString("ZhiXingChengGong")%>";
				} else if ('3' == value){
					return "<%=rb.getString("ZhiXingShiBai")%>";
				}
			},			
            closeConfig() {
                eventBus.$emit('close-config');
            },

            enbAddOrUpdateSubmit() {
                var vm = this, params = {};   
				params.policyId = vm.curType == 'modify' ? vm.curPolicyId : '';
				params.selfStartEnable = vm.addOrEditForm.selfStartEnable;
                params.policyName = encodeURIComponent(vm.addOrEditForm.policyName);
                params.productType = vm.addOrEditForm.productType;
                params.executeType = vm.addOrEditForm.executeType;
                params.upgradeEnable = vm.addOrEditForm.upgradeEnable;               
                //选中 Any 时，参数传 'all'; 未选中时，参数传 ‘初始版本的数据’
                if(vm.addOrEditForm.specifyVersionType == '1'){
                	params.originalVersion = 'all';
                }else{
                	if(vm.resultOriginalVersionList.length > 0){
                		params.originalVersion = vm.resultOriginalVersionList.map(function(item){ return item.originalVersion;}).join(',');
                	}
                }
                params.targetVersion = vm.addOrEditForm.targetVersion;
                params.preserveSetting = vm.addOrEditForm.preserveSetting;
                params.licenseEnable = vm.addOrEditForm.licenseEnable;
				// advance settings params
				for(key in vm.advanceForm) {
					if(vm.paramKeys.includes(key)) {
						params[key] = vm.advanceForm[key];
					}
				}
				params.ipsecList = JSON.stringify(vm.advanceForm.ipsecList);
				params.showGroup = vm.advanceTreeSelected.join(',');
                //参数配置模块参数
                params.platform = vm.curProductName; //产品类型对应的name
                params.selfConfigEnable = vm.addOrEditForm.selfConfigEnable; //参数配置总开关
                params.switchEnable = vm.addOrEditForm.switchEnable; //参数数据池开关
                params.planConfigEnable = vm.addOrEditForm.planConfigEnable; //plan 开关
                params.selfAutoMapEnable = vm.addOrEditForm.selfAutoMapEnable; //tac自动配置开关
                if(vm.addOrEditForm.autoConfigMode == '0'){
                	params.autoConfigMode = 'MAP';
                }else if(vm.addOrEditForm.autoConfigMode == '1'){
                	params.autoConfigMode = 'TAC';
                }else if(vm.addOrEditForm.autoConfigMode == '2'){
                	params.autoConfigMode = 'SN';
                }
                params.defaultTac = vm.addOrEditForm.defaultTac; //enb id-tac 复选框开关
                params.bands_support = vm.addOrEditForm.bands_support;
                params.band_width = vm.addOrEditForm.band_width;
                params.subframe_assignment = vm.addOrEditForm.subframe_assignment;
                params.special_subframe_patterns = vm.addOrEditForm.special_subframe_patterns;
                params.plmn_id = vm.addOrEditForm.plmn_id;
                params.tac = vm.addOrEditForm.tac;
                params.cell_identity = vm.addOrEditForm.cell_identity;
                params.phycellid = vm.addOrEditForm.phycellid;
                params.root_sequence_index = vm.addOrEditForm.root_sequence_index;
                params.mme_ip = vm.addOrEditForm.mmeGroup.toString();
                params.halob_enable = vm.addOrEditForm.halob_enable;
                params.carrier_mode = vm.addOrEditForm.carrier_mode;
				params.custParam = JSON.stringify(vm.addOrEditForm.custParam);

                if(vm.curProductName == "QAFA"){
					params.module_enable = vm.addOrEditForm.module_enable;
					params.moduleTypeList = JSON.stringify(vm.addOrEditForm.moduleTypeList);
				}
				if(vm.curProductName == "DXDF"){
					params.txPower = (vm.addOrEditForm.txPower || []).join(',');
					params.frequency = vm.addOrEditForm.frequency;
				}else{
					params.frequency = vm.addOrEditForm.frequency.substring(0,vm.addOrEditForm.frequency.indexOf("("));
				}
                vm.$refs.addOrEditForm.validate(function(valid){
					if(valid){
						vm.saveBtnDisabled = true;
                        axios.post('${ctx}/SON/SelfConfiguration/addSelfConfigPolicy.action',stringify(params)).then(function(res){
                        	var data = res.data;
                        	if(data){
                        		 vm.saveBtnDisabled = false;
                        		if(data['success'] == true) {
                                    vm.$message({
    		    						message: '<%=rb.getString("ChengGong")%>',
    		    						type:'success',
    		    					});
                                    eventBus.$emit('reload-config-list');
                                    vm.closeConfig();
                                }else {
                                    vm.$message.error(data['message']);
                                }
                        	}
                        }).catch(function(){});
					}
				}) 
            },
            enbAddOrUpdateCancel(){
				eventBus.$emit('cancel-plugPlaySlide')
            },
			createId(idVal,list){
				var vm = this, val = idVal + '';
				if(list.includes(val) == true){
					idVal += 1 ;
					return vm.createId(idVal,list);
				}else{
					return  idVal + '';
				}
			}
        },
        mounted() {
        	var vm = this;
            eventBus.$off('save-plugPlayConfig').$on('save-plugPlayConfig', vm.enbAddOrUpdateSubmit);
            eventBus.$off('handle-plugPlayCancel').$on('handle-plugPlayCancel',vm.enbAddOrUpdateCancel);
            eventBus.$off('init-plugPlayConfig').$on('init-plugPlayConfig', vm.enbInit);
            $("#uploadForm_configPlan input[name='uploadFile']").bind("change", function() {
				vm.importForm.filePath = this.value
			});
        }
    });
</script>
