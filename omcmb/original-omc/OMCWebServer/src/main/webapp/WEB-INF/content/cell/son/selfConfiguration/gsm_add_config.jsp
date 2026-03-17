<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	#gsmAddOrEditConfigPage { background: #F6F7FB; position: relative; width: 100%; height: 100%; overflow: hidden; }
	#gsmAddOrEditConfigPage .gsmWarp { height: calc(100% - 2px); width: 100%; }
	#gsmAddOrEditConfigPage .leftWarp { flex: 1; height: 100%; position: relative; overflow: hidden; flex-direction: column; border-radius: 10px; border: 1px solid #D5DCEC; background: #FFFFFF; }
	#gsmAddOrEditConfigPage .titleBox { width: 100%; height: 38px; line-height: 38px; position: absolute; top: 0; left: 0; z-index: 100; border-bottom: 1px solid #D5DCEC; background: #FFFFFF; }
	#gsmAddOrEditConfigPage .slideTopTitle { padding: 0 30px; }
	#gsmAddOrEditConfigPage .newIconBoxCls-bt { right: 15px; top: 8px; }
	#gsmAddOrEditConfigPage .newIconBoxCls-bt .el-icon-circle-close:before { content: '\e778'; font-size: 12px !important; }
	#gsmAddOrEditConfigPage .mainContent { width: 100%; overflow-y: scroll; height: 100%; position: absolute; top: 40px; z-index: 0; box-sizing: border-box; bottom: 70px; height: auto; }
	#gsmAddOrEditConfigPage .mainContent .el-form-item { margin-bottom: 22px; }
	#gsmAddOrEditConfigPage .mainContent .el-form-item__label { line-height: 28px; }
	#gsmAddOrEditConfigPage .basicInfoBox { margin: 16px 40px 10px; }
	#gsmAddOrEditConfigPage .basicInfo { padding-left: 26px; padding-top: 16px; }
    #gsmAddOrEditConfigPage .split-line { border: none; border-top: 1px solid #D5DCEC; margin: 10px 0 20px 0; }
	#gsmAddOrEditConfigPage .el-radio.is-bordered { max-width: 160px; height: 30px; padding: 7px 12px; }
	#gsmAddOrEditConfigPage .el-radio-group .el-radio__label { font-size: 12px; }
	#gsmAddOrEditConfigPage .enableCommon .el-switch { margin-top: 4px; }
	#gsmAddOrEditConfigPage .selectCommon .el-select .el-input { width: 150px; }
	#gsmAddOrEditConfigPage .el-select .el-input__inner, #gsmAddOrEditConfigPage .el-select .el-input.is-disabled .el-input__inner { height: 26px !important; }
	#gsmAddOrEditConfigPage .inputCommon .el-input__inner { border-radius: 4px; }
	#gsmAddOrEditConfigPage .basicLeftLabel { display: flex; }
	#gsmAddOrEditConfigPage .basicLeftLabel .el-form-item__label { width: 125px; }
	#gsmAddOrEditConfigPage .selectMethodBox { display: flex; flex-direction: row; justify-content: space-between; width: 80%; }
	#gsmAddOrEditConfigPage .selectMethodBox .el-radio-button__inner { display: flex; border: 1px solid #D5DCEC; min-width: 240px; height: 50px; background: #F5F7FE; line-height: 50px; border-radius: 4px; padding: 0; font-size: unset; font-weight: normal; text-align: center; }
	#gsmAddOrEditConfigPage .selectMethodBox .el-radio-button__inner [class*=el-icon-]+span { margin-left: 0; }
	#gsmAddOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .methodTitle { color:var(--main-color); }
	#gsmAddOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .commonIconStyle,
	#gsmAddOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover,
	#gsmAddOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover .commonIconStyle,
	#gsmAddOrEditConfigPage .el-radio:hover, 
	#gsmAddOrEditConfigPage .el-radio .el-radio__inner:hover { color:var(--main-color); background: rgba(var(--main-color-rgba1),0.1); border-color:var(--main-color) !important; }
	#gsmAddOrEditConfigPage .selectMethodBox .el-radio-button__orig-radio:checked+.el-radio-button__inner .el-icon-menu-system:before,
	#gsmAddOrEditConfigPage .selectMethodBox .el-radio-button__inner:hover .el-icon-menu-system:before { color:var(--main-color); }
	#gsmAddOrEditConfigPage .commonIconStyle { margin: 10px 15px; width: 30px; height: 30px; color: #7A7992; background: rgba(122,121,146,0.1); border-radius: 100px; line-height: 30px !important; }
	#gsmAddOrEditConfigPage .originalBox .el-checkbox { padding: 3px 6px 0 10px; }
	#gsmAddOrEditConfigPage .addVersionWarp { width: 400px; height: 254px; border-radius: 4px; text-align: center; border: 1px solid #D5DCEC; }
	#gsmAddOrEditConfigPage .addVersionWarp .el-input.is-disabled .el-input__inner { background-color: #FFFFFF; cursor: pointer; }
	#gsmAddOrEditConfigPage .addVersionBtn { font-size: 20px; padding: 100px 0 10px; display: block; }
	#gsmAddOrEditConfigPage .selectVersionBox .el-form-item__content { margin-left: 0px !important; }
	#gsmAddOrEditConfigPage .selectVersionBox .el-form-item__content .el-input { width: 300px; }
	#gsmAddOrEditConfigPage .commonDisplayBlock { display: block; }
	#gsmAddOrEditConfigPage .operBtn { width: 24px; height: 24px; font-size: 12px; color: #7A7992; line-height: 24px; border: 1px solid #D7D7E6; border-radius: 8px; }
	#gsmAddOrEditConfigPage .importIcon:before,
	#gsmAddOrEditConfigPage .el-icon-menu-system:before { color: #7A7992; }
	#gsmAddOrEditConfigPage .table-suffix { border: none; position: relative; }
	#gsmAddOrEditConfigPage .table-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 2px; color : var(--main-color); font-size: 14px;}
	#gsmAddOrEditConfigPage .table-suffix:hover .deleteVersion { display: inline-block !important; }
	#gsmAddOrEditConfigPage .table-suffix .text { padding-left: 10px; color: #666666; }
	#gsmAddOrEditConfigPage .disabledClass { cursor: not-allowed !important; opacity: 0.4; }
	#gsmAddOrEditConfigPage .disabledClass:before { color: #c0c4cc; cursor: not-allowed !important; }
	#gsmAddOrEditConfigPage .defaultClass { cursor: pointer; }
	#gsmAddOrEditConfigPage .upgradeTitleCls { padding-bottom: 20px; margin-top: 14px; display: block; }
	#gsmAddOrEditConfigPage .selectedVersion .queryGroup .el-input__inner,
	#gsmAddOrEditConfigPage .curImportQuery .el-input__inner { width: 220px !important; }
	#gsmAddOrEditConfigPage .selectedVersion .el-query .advanceQuery { height: 28px; }
	#gsmAddOrEditConfigPage .selectedVersion .el-query .advanceQuery .el-input.el-input--small { width: 230px !important; }
	#gsmAddOrEditConfigPage .selectedVersion .el-query { right: 0px; }
	#gsmAddOrEditConfigPage .footerBox { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; border-top: 1px solid #D5DCEC; justify-content: space-between; display: flex; }
	#gsmAddOrEditConfigPage .footerBox div { padding: 10px 30px 0; }
	#gsmAddOrEditConfigPage .rightBox { position: relative; flex: 0 1 360px; height: 100%; margin-left: 10px; border-radius: 10px; overflow: hidden; border: 1px solid #D5DCEC; background: #FFFFFF; }
	#gsmAddOrEditConfigPage .rightBox .commonRightTitleBox { height: 38px; line-height: 38px; display: flex; justify-content: space-between; margin: 0px 20px; border-bottom: 1px solid #D5DCEC; }
	#gsmAddOrEditConfigPage .rightBox .commonRightTitleBox .el-icon { margin: 12px 0; color:#7A7992; }
	#gsmAddOrEditConfigPage .rightBox .titleAndNoteBox .el-form-item__label { display: flex; }
	#gsmAddOrEditConfigPage .rightHeaderBox { display: flex; height: 38px; line-height: 38px; padding: 0 20px; justify-content: space-between; }
	#gsmAddOrEditConfigPage .rightHeaderBox .addTitle { padding: 0; }
	#gsmAddOrEditConfigPage .rightHeaderBox .closeIconBox .el-icon { font-size: 14px !important; color: #7a7992; margin: 12px 16px; }
	#gsmAddOrEditConfigPage .contentHeight { height: calc(100% - 100px); overflow: scroll; }
	#gsmAddOrEditConfigPage .rightTextBox { padding: 10px 0; border-bottom: 1px solid #D5DCEC; }
	#gsmAddOrEditConfigPage .originalVersionBox { padding: 16px 0 0; }
	#gsmAddOrEditConfigPage .originalVersionBox .el-input{ width: 300px }
	#gsmAddOrEditConfigPage .width280 .el-input{ width: 280px; }
	#gsmAddOrEditConfigPage .originalVersionBox .el-input__inner{ height: 30px; line-height: 30px; }
	#gsmAddOrEditConfigPage .originalVersionBox .el-input-group__append { padding: 0 8px; background-color: #FFFFFF; }
	#gsmAddOrEditConfigPage .originalVersionBox .el-input-group__append .el-icon { font-size: 14px; }
	#gsmAddOrEditConfigPage .originalVersionBox .el-input-group__append .el-icon:before { color: #7A7992; }
	#gsmAddOrEditConfigPage .originalVersionBox .el-table tr { display: none; }
	#gsmAddOrEditConfigPage .ipErrorTip { color: #FA5555; font-size: 12px; }
	#gsmAddOrEditConfigPage .versionResultBox { border-radius: 4px; background-color: #FFFFFF; margin-top: 5px; width: 298px; max-height: 122px; padding: 5px 0; overflow: auto; border: 1px solid #D5DCEC;}
	#gsmAddOrEditConfigPage .versionResultBox .el-form-item { margin-right: 0 !important;}
	#gsmAddOrEditConfigPage .form-suffix { position: relative; margin: 3px 0 0 20px; padding: 0; width:278px; border: none; background: #fff; }
	#gsmAddOrEditConfigPage .form-suffix:hover { background: #F4F9FF; border-radius: 100px; }
	#gsmAddOrEditConfigPage .form-suffix .deleteVersion { display: none; position: absolute; right: 0; top: 0px; color: var(--main-color); font-size: 14px;}
	#gsmAddOrEditConfigPage .form-suffix:hover .deleteVersion { display: inline-block !important; }
	#gsmAddOrEditConfigPage .form-suffix .text { padding-left: 10px; color: #666666; }
	#gsmAddOrEditConfigPage .commonBorder { border: 1px solid #D5DCEC; border-radius: 10px; }
	#gsmAddOrEditConfigPage .commonBorderTop { border-top: 1px solid #D5DCEC; }
	#gsmAddOrEditConfigPage .commonFormFotter { width: 100%; height: 46px; background: #FFFFFF; position: absolute; bottom: 0; left: 0; z-index: 100; }
	#gsmAddOrEditConfigPage .el-form-item__error { padding-top: 0 !important; }
	#gsmAddOrEditConfigPage .licenseBox .el-query{ right: 0; }
	.gsmIpsecConfigAddDialog .licenseImportBox, 
	#gsmAddOrEditConfigPage .licenseImportBox { margin: 0 10px; width: 26px; height: 26px; border: 1px solid #D7D7E6; text-align: center; border-radius: 8px; }
	.gsmIpsecConfigAddDialog .licenseImportBox i, 
	#gsmAddOrEditConfigPage .licenseImportBox i { font-size: 12px; line-height: 26px; }
	.gsmIpsecConfigAddDialog .gsmConfigAddMainBoxCls{ height: 400px;width: 100%;position: relative;overflow-y: auto!important;overflow-x:hidden; }
	.gsmIpsecConfigAddDialog .tieleTipCircle, 
	#gsmAddOrEditConfigPage .tieleTipCircle { width: 8px; margin: 14px 6px 0 0; height: 8px; border-radius: 50%; background: rgba(0, 0, 0, 0.8); display: inline-block; }
	.gsmIpsecConfigAddDialog .contentTableTitle, 
	#gsmAddOrEditConfigPage .contentTableTitle{ display: flex;justify-content: space-between;width: 100%; }
	.gsmIpsecConfigAddDialog .plmnLengthValidate .el-form-item__error {top: unset !important;padding-top: unset !important;}
	.gsmIpsecConfigAddDialog .gsmConfigAddMainBoxCls {height: 400px;width: 100%;position: relative;overflow-y: auto!important;overflow-x:hidden;}
	.gsmIpsecConfigAddDialog .closeAddVlanCls .el-icon-close{font-size: unset;position: unset;top: unset;right: unset;}
	.gsmIpsecConfigAddDialog .inputAndSelect{ position: relative; }
	.gsmIpsecConfigAddDialog .inputAndSelect .el-select>.el-input{ width: 55px!important; }
	.gsmIpsecConfigAddDialog .el-form-item { margin-bottom: 20px; }
	.gsmIpsecConfigAddDialog .inputAndSelect .el-select>.el-input .el-input__inner { width: 55px!important; }
	.gsmIpsecConfigAddDialog .inputAndSelect .el-input .el-input__inner { width: 145px !important; }
	.gsmIpsecConfigAddDialog .inputAndSelect .el-input-group__append { padding-left: 65px!important; }
	.gsmIpsecConfigAddDialog .el-dialog__header .el-icon:before { font-size: 16px; }
	.gsmIpsecConfigAddDialog .el-form { display: flex; flex-wrap: wrap; justify-content: space-between; width: 100%; position: relative; }
	.gsmIpsecConfigAddDialog .el-form-item { display: inline-block; width: 48%; }
	.gsmIpsecConfigAddDialog .el-form-item .el-form-item__label { font-size: 12px; }
	.gsmIpsecConfigAddDialog .el-form-item__error { padding-top: 0px; top:30px!important; }
	.gsmIpsecConfigAddDialog .selectErrcCls .el-form-item__error { position: absolute; left: 210px; top:5px !important; }
	.gsmIpsecConfigAddDialog .validate-item .el-input__inner,
	.gsmImportConfigAddDialog .validate-item .el-input__inner,
	#gsmAddOrEditConfigPage .vlanIdItem .el-input__inner { width:200px; }
	.gsmIpsecConfigAddDialog .validate-item .el-input-group__append,
	.gsmImportConfigAddDialog .validate-item .el-input-group__append,
	#gsmAddOrEditConfigPage .vlanIdItem .el-input-group__append { border:none; background:none; padding: 0px 10px; }
	.gsmIpsecConfigAddDialog .validate-item .el-form-item__error,
	.gsmImportConfigAddDialog .validate-item .el-form-item__error,
	#gsmAddOrEditConfigPage .vlanIdItem .el-form-item__error { display:none; }
	.gsmImportConfigAddDialog .is-error .el-input-group__append { color:#FA5555; }
	.gsmIpsecConfigAddDialog .is-error .el-input-group__append { color:#FA5555; }
	.gsmIpsecConfigAddDialog .el-collapse-item__header { border-bottom:1px solid #fff; }
	.gsmIpsecConfigAddDialog .el-collapse-item__arrow { position:absolute; left:20px; top:0px; }
	.gsmIpsecConfigAddDialog .el-collapse-item { position:relative; border-bottom:1px solid #E9E9E9; }
	.gsmIpsecConfigAddDialog .el-collapse-item__content { margin: 0px 40px; padding-bottom: unset; }
	.gsmIpsecConfigAddDialog .el-collapse-item__header .el-icon-arrow-right { font-size:16px; }
	.gsmIpsecConfigAddDialog .el-collapse-item__header .el-icon-arrow-right:before { content:"\e639"; color:#BBB; }
	.gsmIpsecConfigAddDialog .el-collapse-item__header .is-active.el-icon-arrow-right:before { content:"\e638"; color:#BBB; }
	.gsmIpsecConfigAddDialog .el-collapse-item__arrow.is-active { transform:rotate(0deg); }
	.gsmIpsecConfigAddDialog .el-collapse { border-top:1px solid #fff; border-bottom:1px solid #fff; }
	.gsmIpsecConfigAddDialog .el-collapse { width: 100%; }
	.gsmIpsecConfigAddDialog .el-collapse-item__wrap { border-bottom:1px solid #fff; padding-left: 0px; }
	.gsmIpsecConfigAddDialog .el-collapse-item__header { max-width:800px; }
	.gsmIpsecConfigAddDialog .PlmnListItemCls { border: 1px solid #DFE2EE; position: relative; margin-bottom: 20px; padding: 20px; }
	.gsmIpsecConfigAddDialog .plmnListDelIcon { height: 26px; width: 26px; display: flex; align-items: center; justify-content: center; border: 1px solid #DFE2EE; background-color: #FFFFFF; border-radius: 100px; position: absolute; right:-12px; top:-12px; z-index: 231 }
	.gsmIpsecConfigAddDialog .sliceListBoxCls .el-collapse-item__content { margin: 0px!important; }
	.gsmIpsecConfigAddDialog .sliceListBoxCls .el-collapse-item{ border-bottom: none!important; }
	.gsmIpsecConfigAddDialog .commonParamImportItem { width: 48%; min-width: 400px; }
	.gsmIpsecConfigAddDialog .sliceListBoxCls .plmnItemSliceListCls{ padding: 20px; width: 95%; border: 1px solid #DFE2EE; background-color: #FAFAFD; border-radius: 4px; position: relative; margin-bottom: 20px; }
	.gsmIpsecConfigAddDialog .commonParamBtn, 
	#gsmAddOrEditConfigPage .commonParamBtn { border: 1px solid rgb(213, 220, 236); padding: 0 20px; border-radius: 4px; height: 22px; cursor: pointer; line-height: 22px; }
	.gsmImportConfigAddDialog .titleAndNoteBox .el-form-item__label, 
	#gsmAddOrEditConfigPage .rightBox .titleAndNoteBox .el-form-item__label { display: flex; }
	.gsmImportConfigAddDialog .specialItemCls, 
	#gsmAddOrEditConfigPage .specialItemCls {border: 1px solid #DFE2EE;position: relative;margin: 20px;padding: 10px 20px 0;border-radius: 8px;}
	.gsmImportConfigAddDialog .specialItemDelIcon, 
	#gsmAddOrEditConfigPage .specialItemDelIcon { height: 26px; width: 26px; display: flex; align-items: center; justify-content: center; border: 1px solid #DFE2EE; background-color: #FFFFFF; border-radius: 100px; position: absolute; right:-12px; top:-12px; z-index: 231 }
	.gsmImportConfigAddDialog .el-icon-close { font-size: 16px !important;}
	.gsmImportConfigAddDialog .commonParamImportItem{ width: 40%; min-width: 280px; }
	.gsmImportConfigAddDialog .allowMoreInputAddBtnCls { height: 26px; width: 56px; display: flex; align-items: center; justify-content: center; color:var(--main-color); border: 1px solid var(--main-color); background:rgba(var(--main-color-rgba1),0.1); border-radius: 4px; box-sizing: border-box; margin-left: 10px; cursor: pointer; }
	#gsmAddOrEditConfigPage .paramTwoBox { padding: 10px 20px; }
	#gsmAddOrEditConfigPage .batchParamsContent .el-tabs__content { margin-top: 0px; }
	#gsmAddOrEditConfigPage .paramPoolWarp .el-form-item { width: 49%; display: inline-block; flex-direction: row; }
	#gsmAddOrEditConfigPage .paramPoolWarp .el-form-item .el-input { width: 200px; }
	#gsmAddOrEditConfigPage .paramPoolWarpThree .el-form-item { width: 33%; min-width: 460px; display: inline-block; flex-direction: row; }
	#gsmAddOrEditConfigPage .inputCommon .el-input__inner { border-radius: 4px; }
	#gsmAddOrEditConfigPage .validateItem .el-input-group__append { border:none; background:none; padding: 0px 10px; }
	#gsmAddOrEditConfigPage .is-error .el-input-group__append { color:#FA5555; }
	#gsmAddOrEditConfigPage .validateItem .el-input__inner { width: 200px; }
	#gsmAddOrEditConfigPage .validateItem .el-input-group__append, #gsmAddOrEditConfigPage .validateRightItem .el-input-group__append { border:none; background:none; }
	#gsmAddOrEditConfigPage .validateItem .el-form-item__error, #gsmAddOrEditConfigPage .validateRightItem .el-form-item__error { display:none; }
	#gsmAddOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__arrow { position: absolute; left: 0; top: 0px; }
    #gsmAddOrEditConfigPage .infoSpecifiedDevice .el-collapse-item {  position: relative; }
    #gsmAddOrEditConfigPage .infoSpecifiedDevice .el-icon-arrow-right {  font-size: 16px; }
    #gsmAddOrEditConfigPage .infoSpecifiedDevice .el-icon-arrow-right:before { content: "\e639"; color: #BBB;  }
    #gsmAddOrEditConfigPage .infoSpecifiedDevice .is-active.el-icon-arrow-right:before { content: "\e638"; color: #BBB; }
    #gsmAddOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__arrow.is-active { transform: rotate(0deg); }
    #gsmAddOrEditConfigPage .infoSpecifiedDevice .el-collapse { border-top: 1px solid #fff; border-bottom: 1px solid #fff; }
    #gsmAddOrEditConfigPage .infoSpecifiedDevice .el-collapse-item__wrap { border-bottom: 1px solid #fff; }
    #gsmAddOrEditConfigPage .infoSpecifiedDevice .btn-next .el-icon-arrow-right:before { content: "\e794"; }
	#gsmAddOrEditConfigPage .el-input-group__append, #gsmAddOrEditConfigPage .el-input-group__prepend { color: rgba(0, 0, 0, 0.32); }
	#gsmAddOrEditConfigPage .paramTwoBox .el-collapse-item__content { padding-bottom: 0; }
	#gsmAddOrEditConfigPage .infoImportBox .el-collapse-item__content { padding-bottom: 0; }
	#gsmAddOrEditConfigPage .commonText { display: flex; padding: 3px 0; justify-content: space-between;}
	#gsmAddOrEditConfigPage .marginRight5 { margin-right: 5px; }
	#gsmAddOrEditConfigPage .commonText1 { display: flex; justify-content: space-between; padding: 0 10px; }
	#gsmAddOrEditConfigPage .enableShowOrHide tr:first-child td .el-switch,
	.gsmIpsecConfigAddDialog .enableShowOrHide tr:first-child td .el-switch{
		display: none;
	}
	#gsmAddOrEditConfigPage .infoImportBox .wanDataInfoCls:first-child .enableShowOrHide{
		display: none;
	}
</style>
<div id="gsmAddOrEditConfigPage">
   <div class='commonFlex gsmWarp'>
   		<div class='leftWarp'>
   			<div class='titleBox'>
   				<span class='slideTopTitle commonText14'>{{gsmPolicyTitle}}</span>
   				<div class="newIconBoxCls-bt el-icon el-icon-circle-close" @click="gsmAddOrUpdateCancel"></div>
   			</div>
   			<div class='mainContent'>
   				<el-form ref="gsmAddOrEditForm" :model="gsmAddOrEditForm" :rules="gsmAddOrEditRule" label-position="top" label-width="125" :disabled="gsmOperType == 'view'">
                    <div class="basicInfoBox">
			        	<div class="group-title not-extend"><span class="title-icon"></span><span class="title-text"><%=rb.getString("JiBenXinXi") %></span></div>
						<div class='basicInfo'>
							<el-form-item label='<%=rb.getString("SheZhiKaiGuan") %>' prop='policySwitch' class='enableCommon basicLeftLabel'>
								<el-switch v-model="gsmAddOrEditForm.policySwitch" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
							</el-form-item>
							<el-form-item label='<%=rb.getString("CeLueMingChen") %>' prop="policyName" class='inputCommon basicLeftLabel'>
				                <el-input v-model="gsmAddOrEditForm.policyName" style="width: 300px;"></el-input>
				            </el-form-item>
				            <el-form-item label='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' prop="productType" placeholder='<%=rb.getString("QingXuanZe") %>' class='selectCommon basicLeftLabel'>
				                <el-select v-model="gsmAddOrEditForm.productType" :disabled="gsmOperType != 'add'" @change="gsmProductTypeChange">
				                    <el-option v-for="item in gsmProductList" :label="item.name" :value="item.value"></el-option>
				                </el-select>
				            </el-form-item>
				            <el-form-item label='<%=rb.getString("ZhiXingFangShi") %>' prop="executeType" style='margin-bottom: 30px;' class='basicLeftLabel'>
				                <el-radio-group v-model="gsmAddOrEditForm.executeType">
				                    <el-radio label="0" border><%=rb.getString("eNBZiDongZhiXing") %></el-radio>
				                    <el-radio label="1" border style='margin-left: 20px;'><%=rb.getString("eNBShouDongZhiXing") %></el-radio>
				                </el-radio-group>
				            </el-form-item>
						</div>
					</div>
					<hr class="split-line">
					<div class='basicInfoBox'>
			        	<span class='commonTitle12' style='padding-bottom: 16px; display: block;'><%=rb.getString("KeYiXuanZeYiXiaMoKuaiJinXingPeiZhi") %></span>
			        	<div style="padding-bottom: 6px;">
							<el-form>
								<el-radio-group v-model='gsmFunctionModulesSelect' class='selectMethodBox' @change='gsmSelectMethodChange'>
									<el-radio-button label="0">
										<i class='el-icon el-icon-status-upgrading commonIconStyle'></i><span class='commonTextNormal14 methodTitle'><%=rb.getString("RuanJianShengJi") %></span>
									</el-radio-button>
									<el-radio-button label="1">
										<i class='el-icon el-icon-operation-edit commonIconStyle'></i><span class='commonTextNormal14 methodTitle'>License</span>
									</el-radio-button>
									<el-radio-button label="2">
										<i class='el-icon el-icon-menu-system commonIconStyle'></i><span class='commonTextNormal14 methodTitle'><%=rb.getString("ENBCanShuZiPeiZhi") %></span>
									</el-radio-button>
								</el-radio-group>
							</el-form>
			        	</div>
			        </div>
					<!--upgrade-->
					<div v-show="gsmFunctionModulesSelect == '0'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend">
							<span class="title-icon"></span><span class="title-text"><%=rb.getString("RuanJianShengJi") %></span>
							<el-switch v-model="gsmAddOrEditForm.upgradeEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
						</div>
						<div style='padding-left: 26px;'>
				            <span class='commonTitle12 upgradeTitleCls'><%=rb.getString("NinKeYiShouDongHuoLieBiaoXuanZeChuShiBanBenHao") %></span>
				            <div class='commonFlex originalBox'>
				            	<span class='commonSize14'><%=rb.getString("ChuShiBanBen") %></span>
				                <el-checkbox v-model="specifyVersionType" true-label="1" false-label="0"></el-checkbox>
				            	<span class='commonSize12ExportText' style='margin-top: 1px;'><%=rb.getString("SuoYouAny") %></span>
				            </div>
							<div class='commonFlex' style='padding: 10px 0 4px;'>
								<div class='addVersionWarp' v-show='gsmAddVersionBtnShow'>
									<div v-if="specifyVersionType == '1' || gsmOperType == 'view'" class='el-icon el-icon-plus addVersionBtn disabledClass'></div>
									<div v-else @click='gsmAddVersionClick' class='el-icon el-icon-plus addVersionBtn defaultClass'></div>
									<span class='commonTitle12'><%=rb.getString("NinKeYiTianJiaYuanShiBanBen") %></span>
								</div>
								<div class='addVersionWarp' v-show='gsmResultOriginalVersionShow'>
									<el-ctable ref="resultOriginalVersionTable" row-key="originalVersion" style='margin-bottom: 10px;border-radius: 4px;'
										:data="gsmResultOriginalVersionList.filter(item=>{
											return item.originalVersion.indexOf(gsmSearchValue) > -1
										})" :pagination="false" :rownumber="false">
										<div slot="toolbar">
											<div class='commonFlex selectedVersion'>
												<div class='queryGroup' style='margin-right: 10px;'>
													<el-form style="display: flex;">
														<el-input v-model="gsmSearchValue" class='pairgrid-query' placeholder='<%=rb.getString("ChuShiBanBen")%>' style='width: 220px;'></el-input>
														<i class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
													</el-form>
												</div>
												<div v-show="gsmOperType != 'view' && specifyVersionType == '0'" class="commonFlex">
													<div class='el-icon el-icon-operation-clear operBtn' @click='gsmClearVersionBtnClick' style='margin-right: 10px;'></div>
													<div class='el-icon el-icon-plus operBtn' @click='gsmAddVersionClick' style='margin-right: 10px;'></div>
												</div>
											</div>
										</div>
 										<el-table-column label='<%=rb.getString("ChuShiBanBen") %>' prop="originalVersion" show-overflow-tooltip="true">
											<template slot-scope="scope">
												<div class='table-suffix'>
						                        	<span class="text">{{scope.row.originalVersion}} </span>
						                        	<span v-show="gsmOperType != 'view' && specifyVersionType == '0'"><i class='el-icon el-icon-circle-close deleteVersion' @click='gsmDeleteVersionItem(scope.row)'></i></span>
						                        </div>
											</template>
										</el-table-column>
									</el-ctable>
								</div>
								<div style='padding: 60px 60px 0;'>
									<el-form-item label='<%=rb.getString("MuBiaoBanBen") %>' label-position="top" prop="targetVersion" class="selectVersionBox" style=" margin-bottom: 16px;">
					                    <el-select v-model="gsmAddOrEditForm.targetVersion" placeholder='<%=rb.getString("QingXuanZe") %>'>
					                       <el-option v-for="item in gsmTarVersionList" :key="item.text" :label="item.text" :value="item.text"></el-option>
					                    </el-select>
					                </el-form-item>
					                <el-checkbox v-model="gsmAddOrEditForm.preserveSetting" true-label="1" false-label="0"></el-checkbox>
					                <span class='commonSize14' style='margin-left: 6px;'><%=rb.getString("eNBBaoLiuPeiZhi") %></span>
								</div>
							</div>
							<el-form-item label-width='0' prop="originalVersion"><el-input v-model="gsmAddOrEditForm.originalVersion" v-show=false></el-input></el-form-item>
						</div>
			        </div>
					<!--license-->
					<div v-show="gsmFunctionModulesSelect == '1'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
			            <div class="group-title not-extend">
							<span class="title-icon"></span><span class="title-text">License</span>
							<el-switch v-model="gsmAddOrEditForm.licenseEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
						</div>
			            <div style='height: 334px; padding-left: 26px;'>
							<el-ctable :height="height" ref="licenseTable" id='licenseTable' :url="gsmLicenseUrl" :time="6" :row-key="'serial_number'" class='commonBorder' style='margin-top: 10px; margin-bottom: 10px;' :pagination="true" :query-params="gsmLicenseQuery" row-key="serial_number">
								<div slot="toolbar">
									<div class='commonFlex commonToolBarBox licenseBox' style="justify-content: space-between; ">
										<span class='commonTitle12' style='padding: 6px 20px;'><%=rb.getString("DaoRuLicenseFile")%></span>
										<div class='commonFlex' style="padding-right: 10px;">
											<el-form  style="display: flex;">
												<el-query type="normal" @query="queryGsmLicense" placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-query>
												<div v-show='gsmOperType != "view"' class='licenseImportBox' @click='gsmImportLicenseClick'><i class='el-icon el-icon-operation-import importIcon'></i></div>
											</el-form>
										</div>
									</div>
								</div>
								<el-table-column width="40" v-if='gsmOperType != "view"'>
									<template slot-scope="scope">
										<div><i @click="gsmDeleteLicenseClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon"></i></div>
									</template>
								</el-table-column>
								<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
								<el-table-column label='<%=rb.getString("LicenseWenJian")%>' prop="file_name"></el-table-column>
								<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' prop="upload_time"></el-table-column>
								<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="execute_status" :formatter="executeStatus"></el-table-column>
							</el-ctable>
			            </div>
			        </div>
					<!--parameter configuration-->
					<div v-show="gsmFunctionModulesSelect == '2'" class="basicInfoBox commonColor" style='margin-top: 20px;'>
						<div class="group-title not-extend">
							<span class="title-icon"></span><span class="title-text"><%=rb.getString("ENBCanShuZiPeiZhi") %></span>
							<el-switch v-model="gsmAddOrEditForm.selfConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
						</div>
						<div style='padding-left: 26px; padding-top: 16px; '>
                            <span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 20px; margin-top: -6px;'><%=rb.getString("GSMCanShuPeiZhiTiShi") %></span>
                            <div class='reginDeployingBox infoSpecifiedDevice' style=' width: 100%; padding-bottom: 100px;'>
                                <el-tabs v-model="gsmParameterConfigActive" @tab-click="gsmClickTab" type='border-card' style='height: auto' class="batchParamsContent">
									<el-tab-pane label='<%=rb.getString("CanShuShuJuChi") %>' name="gsmParamsConfig" style="position:relative">
										<div class='paramTwoBox'>
											<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='paramConfigEnable' label-width='80px' class="enableCommon basicLeftLabel" style='margin-bottom: 0;'>
												<el-switch v-model="gsmAddOrEditForm.paramConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
											</el-form-item>
											<el-collapse v-model="gsmActiveName">
												<el-collapse-item name="basic">
													<template slot='title'><p style="display:inline-block;margin-left: 26px;"> <span class='commonText14'><%=rb.getString("GSMJiChuXinXi")%></span> </p></template>
													<div style='margin-left: 26px; margin-right: 0px;'>
														<div style='width: 100%;' class='paramPoolWarp'>
															<el-form-item label='IpaUnitId' prop="ipaUnitid" class='inputCommon validateItem'>
																<el-input v-model.trim="gsmAddOrEditForm.ipaUnitid">
																	<template slot="append"><%=rb.getString("FanWei")%>: Ipa[0~65535],Id[0~255]</template>
																</el-input>
															</el-form-item>
															<el-form-item label='OmlRemoteIp' prop="omlRemoteIp" class='inputCommon validateItem'>
																<el-input v-model.trim="gsmAddOrEditForm.omlRemoteIp"></el-input>
															</el-form-item>
															<el-form-item label='OmlRemoteIpBak' prop="omlRemoteIpBak" class='inputCommon validateItem'>
																<el-input v-model.trim="gsmAddOrEditForm.omlRemoteIpBak"></el-input>
															</el-form-item>
															<el-form-item label='<%=rb.getString("GSMShePinGongLv")%>' prop="rfPower" class='inputCommon validateItem'>
																<el-input v-model.trim="gsmAddOrEditForm.rfPower">
																	<template slot="append"><%=rb.getString("FanWei")%>: 24~43</template>
																</el-input>
															</el-form-item>
														</div>
													</div>
												</el-collapse-item>
												<el-collapse-item name="advance">
													<template slot='title'>
														<p style="display:flex;margin-left: 26px;"> 
															<span class='commonText14'><%=rb.getString("Advance")%></span>
															<el-form-item label='' prop='advanceEnable' label-width='80px' style="margin: 15px 10px;">
																<el-switch onclick="event.stopPropagation()" v-model="gsmAddOrEditForm.advanceEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
															</el-form-item>
														</p>
													</template>
													<div style='padding: 5px 0px 0 26px;' class='paramPoolWarpThree'>
														<el-form-item label='<%=rb.getString("GSMDNS1")%>' prop="dns1" class='inputCommon validateItem'>
															<el-input v-model.trim="gsmAddOrEditForm.dns1"></el-input>
														</el-form-item>
														<el-form-item label='<%=rb.getString("GSMDNS2")%>' prop="dns2" class='inputCommon validateItem'>
															<el-input v-model.trim="gsmAddOrEditForm.dns2"></el-input>
														</el-form-item>
														<el-form-item label='<%=rb.getString("GNBShiQu")%>' prop="localTimezoneName" class='inputCommon'>
															<el-select v-model='gsmAddOrEditForm.localTimezoneName' filterable>
																<el-option v-for="item in timeZoneList" :key="item.value" :label="item.label" :value="item.value"></el-option>
															</el-select>
														</el-form-item>
													</div>
													<!--Route-->
													<div>
														<div class='commonFlex commonContent' style="padding: 5px 0 0 26px;">
															<div>
																<span class='tieleTipCircle'></span>
																<span class='commonText14'><%=rb.getString("GSMLuYou")%></span>		
																<span class='commonNotes12' style="margin-left: 10px"><%=rb.getString("GSMXianZhiTiaoShu16")%></span>
															</div>
															<div v-if="gsmOperType != 'view' && gsmAddOrEditForm.staticRoute.length < 16" class='licenseImportBox' @click="commonAddOrUpdateParamClick('', 'add', 'router', '')" style="margin: 0;">
																<i class='el-icon el-icon-plus importIcon'></i>
															</div>
														</div>
														<div style='padding: 5px 0px 20px 26px;' class='paramPoolWarp'>
															<el-ctable ref="staticListTable" :rownumber="true" id="staticListTable" :data="gsmAddOrEditForm.staticRoute" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
																<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" v-if="gsmOperType != 'view'">
																	<template slot-scope="scope">
																		<span class="el-icon el-icon-operation-edit" @click="commonAddOrUpdateParamClick(scope.row, 'edit', 'router', '')" style="margin-right:15px;"></span>
																		<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'router', '')" ></span>
																	</template>
																</el-table-column>
																<el-table-column label='<%=rb.getString("SheZhiKaiGuan")%>' prop="onboot" width='100px' show-overflow-tooltip>
																	<template slot-scope="scope">
																		<el-switch onclick="event.stopPropagation()" v-model="scope.row.onboot" active-value="1" inactive-value="0"></el-switch>
																	</template>
																</el-table-column>
																<el-table-column label='<%=rb.getString("WangGuan")%>' prop="gateway" show-overflow-tooltip></el-table-column>
																<el-table-column label='<%=rb.getString("MuDiWangLuo")%>' prop="netAddr" show-overflow-tooltip></el-table-column>
																<el-table-column label='<%=rb.getString("ZiWangYanMa")%>' prop="netMask" show-overflow-tooltip></el-table-column>
															</el-ctable>
															<el-form-item prop='staticRoute' style="display:none;" label="" label-width="0px">
																<el-input v-model='gsmAddOrEditForm.staticRoute'></el-input>
															</el-form-item>
														</div>
													</div>
													<!--WAN Config-->
													<div>
														<div class='commonFlex commonContent' style="padding: 5px 0 0 26px;">
															<div>
																<span class='tieleTipCircle'></span>
																<span class='commonText14'>WAN Config</span>		
																<span class='commonNotes12' style="margin-left: 10px"><%=rb.getString("GSMXianZhiTiaoShu12")%></span>
															</div>
															<div v-if="gsmOperType != 'view' && gsmAddOrEditForm.wan.length < 12" class='licenseImportBox' @click="commonAddOrUpdateParamClick('', 'add', 'wan', '')" style="margin: 0;">
																<i class='el-icon el-icon-plus importIcon'></i>
															</div>
														</div>
														<div style='padding: 5px 0px 20px 26px;' class='paramPoolWarp'>
															<el-ctable ref="staticListTable" :rownumber="true" id="staticListTable" :data="gsmAddOrEditForm.wan" height="200px" :pagination="false" class="enableShowOrHide" style="border:1px solid #E9E9E9;">
																<el-table-column label='<%=rb.getString("CaoZuo")%>' width="100" v-if="gsmOperType != 'view'">
																	<template slot-scope="scope">
																		<span class="el-icon el-icon-operation-edit" @click="commonAddOrUpdateParamClick(scope.row, 'edit', 'wan', '')" style="margin-right:15px;"></span>
																		<span class="el-icon el-icon-operation-delete" @click="commonDelParamClick(scope.row, 'wan', '')" ></span>
																	</template>
																</el-table-column>
																<!--第一行数据中的开关不显示-->
																<el-table-column label='<%=rb.getString("SheZhiKaiGuan")%>' prop="enable" width='100px' show-overflow-tooltip>
																	<template slot-scope="scope">
																		<el-switch onclick="event.stopPropagation()" v-model="scope.row.enable" active-value="1" inactive-value="0"></el-switch>
																	</template>
																</el-table-column>
																<el-table-column label='<%=rb.getString("gsmIPFangWenFangShi")%>' prop="ipMode" min-width="100" show-overflow-tooltip>
																	<template slot-scope="scope">
																		<div v-if="scope.row.ipMode == '0'">DHCP</div>
																		<div v-if="scope.row.ipMode == '1'">Static</div>
																	</template>
																</el-table-column>
																<el-table-column label='<%=rb.getString("IPDiZhi")%>' prop="ipAddr" min-width="120" show-overflow-tooltip></el-table-column>
																<el-table-column label='<%=rb.getString("ZiWangYanMa")%>' prop="netMask" min-width="200" show-overflow-tooltip></el-table-column>
																<el-table-column label='<%=rb.getString("WangGuan")%>' prop="gateway" min-width="120" show-overflow-tooltip></el-table-column>
																<el-table-column label='VLAN ID' min-width="80" prop="vlanId" show-overflow-tooltip></el-table-column>
															</el-ctable>
															<el-form-item prop='wan' style="display:none;" label="" label-width="0px">
																<el-input v-model='gsmAddOrEditForm.wan'></el-input>
															</el-form-item>
														</div>
													</div>
												</el-collapse-item>
												<!--Customized Paeameters-->
												<el-collapse-item name="customized">
													<template slot='title'>
														<div style="display: flex;margin-left: 26px;">
															<span class='commonText14'><%=rb.getString("ZiDingYiCanShu")%></span>
															<div v-if="gsmOperType != 'view'" class='licenseImportBox' @click="addNrPlmnAddClick" style="margin: 10px 10px 0">
																<i class='el-icon el-icon-plus importIcon'></i>
															</div>
														</div>
													</template>
													<div style='margin-left: 26px; margin-right: 0px;' v-show="gsmAddOrEditForm.custParam.length > 0">
														<div style='width: 100%;' class='paramPoolWarpThree'>
															<div v-for="(item,index) in gsmAddOrEditForm.custParam" class="specialItemCls">
																<div style="display:flex;flex-wrap: wrap;font-size:12px">
																	<el-form-item label="ID" :prop="'custParam['+index+'].id'" class='validate-item' v-if="false"> 
																		<el-input v-model.trim='item.id'></el-input>
																	</el-form-item>
																	<el-form-item label='<%=rb.getString("MingCheng")%>' :prop="'custParam['+index+'].custParamName'" class='validate-item'> 
																		<el-input v-model.trim='item.custParamName'></el-input>
																	</el-form-item>
																	<el-form-item label='<%=rb.getString("GNBZhi")%>' :prop="'custParam['+index+'].custParamValue'" class='validate-item'> 
																		<el-input v-model.trim='item.custParamValue'></el-input>
																	</el-form-item>
																	<el-form-item label='<%=rb.getString("GNBLuJing")%>' :prop="'custParam['+index+'].custParamPath'"  class='validate-item' > 
																		<el-input v-model.trim='item.custParamPath' style="min-width: 250px;"></el-input>
																	</el-form-item>
																</div>															
																<div class="specialItemDelIcon" v-if="gsmOperType != 'view'">
																	<span class="el-icon el-icon-circle-close" @click="addNrPlmnListDel(item)" style="font-size: 12px;"></span>
																</div>
															</div>
															<el-form-item prop='custParam' style="display:none;" label-width="0px">
																<el-input v-model='gsmAddOrEditForm.custParam'></el-input>
															</el-form-item>
														</div>
													</div>
												</el-collapse-item>
											</el-collapse>
										</div>
									</el-tab-pane>
									<el-tab-pane label='<%=rb.getString("ZhiDingSheBeiJiHua")%>' name="gsmParamsImport">
                                        <div class='paramTwoBox'>
                                            <el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop='batchConfigEnable' label-width='80px' class="enableCommon basicLeftLabel" style='margin-bottom: 16px;'>
                								<el-switch v-model="gsmAddOrEditForm.batchConfigEnable" active-value="1" inactive-value="0" active-color="#4D84FF" inactive-color="#BDC1C6"></el-switch>
                                            </el-form-item>
											<div style='height: 300px;'>
												<el-ctable ref="paramsImportFileTable" id='paramsImportFileTable' :url="gsmUrlConfigPlan" :time="6" :query-params="importParamsQuery" :row-key="'id'" class='commonBorderRadius commonBorder' style='margin-top: 10px; margin-bottom: 10px;'
													:height="height" pagination="true" rownumber="true">													
													<template slot="toolbar">
														<div class='commonFlex commonToolBarBox licenseBox' style="justify-content: space-between; ">
															<span class='commonText14' style='padding: 6px 20px;'><%=rb.getString("ZhiDingCanShuLieBiao")%></span>
															<div class='commonFlex' style="padding-right: 10px;">
																<el-form  style="display: flex;">
																	<el-query type="normal" @query="gsmBatchImportQuery" placeholder='<%=rb.getString("XiaoZhanBianMa")%>'></el-query>
																	<div v-show='gsmOperType !="view"' class='licenseImportBox' @click='gsmParamImportFileClick'><i class='el-icon el-icon-operation-import importIcon'></i></div>
																</el-form>
																<div v-if='false' class='licenseImportBox' @click='gsmParamExportClick'><i class='el-icon el-icon-operation-export importIcon'></i></div>
															</div>
														</div>
													</template>
													<el-table-column width="100">
														<template slot-scope="scope">
															<div>
																<i v-show='gsmOperType != "view"' @click="paramsImportTableEditClick(scope.row, event)" class="el-icon el-icon-operation-edit commonIcon"></i>
																<i v-show='gsmOperType != "view"' @click="paramsImportTableDelClick(scope.row, event)" class="el-icon el-icon-operation-delete commonIcon" style='margin-left: 6px;'></i>
																<i @click="paramsImportTableViewClick(scope.row, event)" class="el-icon el-icon-operation-info commonIcon" style='margin-left: 6px;'></i>
															</div>
														</template>
													</el-table-column>
													<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serialNumber" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='IpaUnitId' prop="ipaUnitid" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='OmlRemoteIp' prop="omlRemoteIp" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='OmlRemoteIpBak' prop="omlRemoteIpBak" show-overflow-tooltip="true"></el-table-column>
													<el-table-column label='<%=rb.getString("GSMShePinGongLv")%>' prop="rfPower" show-overflow-tooltip="true"></el-table-column>
												</el-ctable>												
								            </div>
										</div>
									</el-tab-pane>	
								</el-tabs>
							</div>
						</div>
					</div>
                </el-form>
   			</div>
		    <div class='footerBox' v-show="gsmOperType !='view'">
   				<div>
   					<el-button type="primary" size="mini" @click='gsmAddOrUpdateSubmit' :disabled='gsmSaveBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='gsmAddOrUpdateCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>

		<!--software upgrade: Add Version  -->
		<div class='rightBox' v-show='gsmAddVersionShow' style="flex: 0 1 340px;">
			<div class='rightHeaderBox'>
   				<span class='AddTitle commonText14'><%=rb.getString("TianJianBanBen")%></span>
   				<span class='closeIconBox' @click='gsmAddVersionCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 10px 0 20px;'>
   				<div class='rightTextBox'><span class='commonTitle12'><%=rb.getString("ShouDongHuoXuanZeBanBen")%></span></div>
   				<div class='originalVersionBox'>
   				   	<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '><%=rb.getString("ChuShiBanBen") %></span>
   					<el-form ref="gsmSoftwareAddVersionForm" :inline="true" label-position="top" :model="gsmSoftwareAddVersionForm" style="display: flex; flex-direction: column;overflow: hidden;">
	   					<el-form-item style='margin-bottom: 16px;'>
							<el-form-item prop='versionStr'>
								<el-input v-model='gsmSoftwareAddVersionForm.versionStr'><i slot="append" class="el-icon el-icon-plus" @click="gsmAddVersionBtn"></i></el-input>
							</el-form-item>
							<div class='versionResultBox' v-show='gsmSoftwareAddVersionForm.versionList.length > 0'>
								<el-form-item class='suffixItem' v-for='(domain,index) in gsmSoftwareAddVersionForm.versionList'>
									<div class='form-suffix'>
										<span class='text'>{{domain}}</span>
										<span class='form-bt-remove el-icon el-icon-circle-close ipTextWarp deleteVersion' @click.prevent='gsmDeleteVersion(domain)'></span>
									</div>
								</el-form-item>
								<el-form-item prop='itemTest'>
									<el-input v-model='gsmSoftwareAddVersionForm.itemTest' v-show=false></el-input>
								</el-form-item>
							</div>
							<p class='ipErrorTip'>{{gsmVersionErrorMessage}}</p>
	                    </el-form-item>
   					</el-form>
   				</div>
   				<div>
	                <span class='commonTitle12 commonDisplayBlock'><%=rb.getString("ChuShiBanBenLieBiao")%></span>
					<el-ctable ref="gsmSoftwareOriginalVersionTable" height='280px' style='margin-top: 8px;border: 1px solid #D5DCEC; border-radius: 4px;' :pagination="false" :rownumber="false"
						:data="gsmSoftwareOriginalVersionList" row-key="originalVersion" @selection-change='gsmVersionBatchSelect'>
						<el-table-column type="selection" :reserve-selection="true"></el-table-column>
						<el-table-column label='<%=rb.getString("ChuShiBanBen") %>' prop="originalVersion" show-overflow-tooltip="true"></el-table-column>
					</el-ctable>
	            </div>
	            <p class='ipErrorTip' style='margin-top: 2px;'>{{gsmVersionMessage}}</p>
   			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" size="mini" @click='gsmAddVersionSubmit'><%=rb.getString("QueDing")%></el-button>
					<el-button size="mini" @click='gsmAddVersionCancel'><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
		<div class='rightBox' style='position: relative;' v-show="gsmCommonAddOrUpdateParamShow">
			<div class='commonRightTitleBox'>
   				<span class='AddTitle commonText14'>{{gsmCommonAddOrUpdateParamTitle}}</span>
   				<span class='el-icon el-icon-close' @click='commonAddOrUpdateParamCancel' style="font-size: 12px; margin: 12px 20px;"></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;'>
   				<div class='originalVersionBox' >
					<el-form ref="gsmCommonAddEditForm" :model='gsmCommonAddEditForm' :rules='gsmCommonAddEditFormRules' label-position="top">     		     			            
						<div v-if="gsmCommonAdvanceOperModel == 'router'">
							<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop="onboot">
								<el-switch onclick="event.stopPropagation()" v-model="gsmCommonAddEditForm.onboot" active-value="1" inactive-value="0"></el-switch>
							</el-form-item>	
							<el-form-item label='<%=rb.getString("WangGuan")%>' prop='gateway'>
								<el-input v-model.trim='gsmCommonAddEditForm.gateway'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("MuDiWangLuo")%>' prop='netAddr'>
								<el-input v-model.trim='gsmCommonAddEditForm.netAddr'></el-input>
							</el-form-item>
							<el-form-item label='<%=rb.getString("GNBZiWangYanMa")%>' prop='netMask'>
								<el-input v-model.trim='gsmCommonAddEditForm.netMask'></el-input>
							</el-form-item>
						</div>
						<div v-if="gsmCommonAdvanceOperModel == 'wan'">
							<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop="enable">
								<el-switch onclick="event.stopPropagation()" v-model="gsmCommonAddEditForm.enable" active-value="1" inactive-value="0"></el-switch>
							</el-form-item>	
							<el-form-item label='<%=rb.getString("gsmIPFangWenFangShi")%>' prop='ipMode'>
								<el-select v-model='gsmCommonAddEditForm.ipMode'>
									<el-option label='DHCP' value='0'></el-option>
									<el-option label='Static' value='1'></el-option>
								</el-select>
							</el-form-item> 
							<div v-if="gsmCommonAddEditForm.ipMode == '1'">
								<el-form-item label='<%=rb.getString("IPDiZhi")%>' prop='ipAddr'>
									<el-input v-model.trim='gsmCommonAddEditForm.ipAddr'></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("GNBZiWangYanMa")%>' prop='netMask'>
									<el-input v-model.trim='gsmCommonAddEditForm.netMask'></el-input>
								</el-form-item>
								<el-form-item label='<%=rb.getString("WangGuan")%>' prop='gateway'>
									<el-input v-model.trim='gsmCommonAddEditForm.gateway'></el-input>
								</el-form-item>
							</div>
							<el-form-item label="VLAN ID" prop='vlanId' style="margin-bottom: 10px;" class='vlanIdItem'> 
								<el-input v-model.trim='gsmCommonAddEditForm.vlanId'>
									<template slot="append"><%=rb.getString("FanWei")%>: 1~4094</template>
								</el-input>
							</el-form-item> 
						</div>
					</el-form>  
   				</div>
			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="commonAddOrUpdateParamSubmit" :disabled='commonAddOrUpdateBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="commonAddOrUpdateParamCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
		<!--import license or batch Import-->
		<div class='rightBox' style='position: relative;' v-show='gsmCommonImportShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14'>{{commonImportTitle}}</span>
   				<span class='closeIconBox' @click='gsmCommonImportCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent contentHeight' style='padding: 0 20px;' v-show="commonImportType == 'license'">
   				<div class='rightTextBox'><span class='commonTitle12'><%=rb.getString("DaoRuLicenseTiShi")%></span></div>
   				<div class='originalVersionBox'>
					<span class='commonTitle12 commonDisplayBlock' style='padding-bottom: 6px; '><%=rb.getString("WenJianMing")%>(<span class='commonTitle12'><%=rb.getString("LicenseGeShi")%></span>)</span>
   					<el-form label-position="top" ref="gsmLicenseForm" :model='gsmLicenseForm' :rules='gsmImportLicenseRules' v-loading="gsmLicenseLoading">
	                  	<el-form-item label=" " prop="fileName">
							<el-upload ref="moreUpload" :multiple="true" :on-success='gsmMoreCheckFile' :on-change="gsmMoreFileChange" :show-file-list=false :action="gsmLicenseForm.uploadFileUrl" 
								:data="gsmMoreFileParams" name="uploadFile" :file-list="gsmMoreFileList" :http-request="gsmMoreFileRequest" :auto-upload="false" accept=".lic"> 
								<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
									<a slot="append" class="el-icon el-icon-operation-import importBox" @click="gsmMoreFileSelect"></a>
								</el-input>
								<a slot="trigger" ref="more_file_up"></a>
							</el-upload>
	                    </el-form-item>
		            </el-form>
   				</div>
   			</div>
			<div class='rightContent contentHeight' style='padding: 0 20px;' v-show="commonImportType == 'batchParams'">
				<div class='rightTextBox commonBorderBottom'><span class='commonTitle12'><%=rb.getString("GNBDaoRuTiShi")%></span></div>
				<div class='originalVersionBox'>
					<el-form ref="gsmImportRuleForm" :model="gsmImportRuleForm" :rules="gsmImportRules" label-position="top" v-loading="commonImportFileLoading">
						<el-form-item label='<%=rb.getString("DaoRuLeiXing")%>' style="margin-bottom: 20px;">
							<el-radio-group v-model="gsmImportRuleForm.operType">
								<el-radio label="0" border><%=rb.getString("ZhuiJia")%></el-radio>
								<el-radio label="1" border><%=rb.getString("TiHuan")%></el-radio>
							</el-radio-group>
						</el-form-item>
						<el-form-item label='<%=rb.getString("WenJian")%>' prop="fileName">
							<el-upload ref="upload" :before-upload='gsmBeforeUpload' :on-success='gsmCheckFile' :on-change="gsmFileChange" :show-file-list=false 	                  		
								:action="gsmImportRuleForm.uploadFileUrl" :data="fileParams" name="uploadFile" accept=".xls,.xlsx" :auto-upload="false">
								<el-input :readonly="true" :value=fileName placeholder='<%=rb.getString("QingXianXuanZeWenJian")%>'>
									<a slot="append" class="el-icon el-icon-operation-import importBox" @click="gsmFileSelect"></a>
								</el-input>								
								<a slot="trigger" ref="file_up"></a>
							</el-upload>
						</el-form-item>
						<div>
							<span class='commonNotes12' style='padding-bottom: 10px;display: block;'><%=rb.getString("DaoRuWenJianTiShi")%></span>
							<span style="cursor:pointer;" @click="gsmExportTemplate">
								<span class='el-icon el-icon-common-download'></span>
								<span class='commonGeneral12' style='text-decoration:underline'><%=rb.getString("DaoChuMuBan")%></span>
							</span>
						</div>
					</el-form>
				</div>
			</div>
   			<div class='commonFlex commonBorderTop commonFormFotter'>
   				<div style='padding: 10px 30px 0;'>
   					<el-button type="primary" @click="gsmCommonImportSubmit" :disabled='gsmCommonUploadBtnDisabled'><%=rb.getString("QueDing")%></el-button>
					<el-button @click="gsmCommonImportCancel"><%=rb.getString("QuXiao")%></el-button>
   				</div>
   			</div>
   		</div>
		<!--batch import failure dailog-->
		<el-dialog title='<%=rb.getString("XinXi")%>' width="400px" :visible="downloadVisible" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeDownloadDialog">
			<div>
				<p style="color:#797979;font-size:14px;"><%=rb.getString("BuFenShuJuCuoWuQingChongXingBianJi")%></p>
				<el-button style="margin-top:20px;margin-left:250px;" @click="downloadErrorFile" type="primary"><%=rb.getString("XiaZai")%></el-button>
			</div>
		</el-dialog>

		<!--batch import: 修改 第一层弹窗-->
		<el-dialog class="gsmIpsecConfigAddDialog" title='<%=rb.getString("XiuGai")%>' top="20vh" width="60%" 
			:visible.sync="paramsImportEditDialogShow" @close="paramsImportEditDialogClose" :close-on-click-modal="false" append-to-body>
			<div class="gsmConfigAddMainBoxCls">
				<el-form ref="gsmImportAddOrEditForm" :model="gsmImportAddOrEditForm" :rules="gsmImportAddOrEditRules" label-position="top" label-width="125">
					<el-collapse v-model="gsmImportActiveName">
						<el-collapse-item name="basic">
							<template slot='title'><p style="display:inline-block;margin-left: 40px;"><span class='commonText14'><%=rb.getString("GSMJiChuXinXi")%></span></p></template>
							<div class="rightContentCls">
								<div style='display:flex;flex-wrap: wrap'>
									<el-form-item label='IpaUnitId' prop="ipaUnitid" class='inputCommon validate-item commonParamImportItem'>
										<el-input v-model.trim="gsmImportAddOrEditForm.ipaUnitid">
											<template slot="append"><%=rb.getString("FanWei")%>: Ipa[0~65535],Id[0~255]</template>
										</el-input>
									</el-form-item>
									<el-form-item label='OmlRemoteIp' prop="omlRemoteIp" class='inputCommon validate-item commonParamImportItem'>
										<el-input v-model.trim="gsmImportAddOrEditForm.omlRemoteIp"></el-input>
									</el-form-item>
									<el-form-item label='OmlRemoteIpBak' prop="omlRemoteIpBak" class='inputCommon validate-item commonParamImportItem'>
										<el-input v-model.trim="gsmImportAddOrEditForm.omlRemoteIpBak"></el-input>
									</el-form-item>
									<el-form-item label='<%=rb.getString("GSMShePinGongLv")%>' prop="rfPower" class='inputCommon validate-item commonParamImportItem'>
										<el-input v-model.trim="gsmImportAddOrEditForm.rfPower" maxlength="150">
											<template slot="append"><%=rb.getString("FanWei")%>: 24~43</template>
										</el-input>
									</el-form-item>
								</div>
							</div>
						</el-collapse-item>
						<!--Advance-->
						<el-collapse-item name="advance">
							<template slot='title'><p style="display:inline-block;margin-left: 40px;"><span class='commonText14'><%=rb.getString("Advance")%></span></p></template>
							<div class="rightContentCls">
								<div>
									<div class='commonFlex' style="padding: 0 0 0 0px; justify-content: space-between;">
										<div>
											<span class='tieleTipCircle'></span>
											<span class='commonText14'><%=rb.getString("GSMLuYou")%></span>		
											<span class='commonNotes12' style="margin-left: 10px"><%=rb.getString("GSMXianZhiTiaoShu16")%></span>
										</div>
										<div v-if="gsmOperType != 'view' && gsmImportAddOrEditForm.staticRoute.length < 16" class='licenseImportBox' @click="importParamClick('', 'add', 'importRoute', '')" style="margin: 0;">
											<i class='el-icon el-icon-plus importIcon'></i>
										</div>	
									</div> 
									<div style='padding: 5px 0px 20px 0px;' class='paramPoolWarp'>
										<el-ctable ref="importRouteListTable" :rownumber="true" id="importRouteListTable" :data="gsmImportAddOrEditForm.staticRoute" height="200px" :pagination="false" style="border:1px solid #E9E9E9;">
											<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
												<template slot-scope="scope">
													<span class="el-icon el-icon-operation-edit" @click="importParamClick(scope.row, 'edit', 'importRoute', '')" style="margin-right:15px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="importParamDelClick(scope.row, 'importRoute', '')" ></span>
												</template>
											</el-table-column>
											<el-table-column label='<%=rb.getString("SheZhiKaiGuan")%>' prop="onboot" width='100px' show-overflow-tooltip>
												<template slot-scope="scope">
													<el-switch onclick="event.stopPropagation()" v-model="scope.row.onboot" active-value="1" inactive-value="0"></el-switch>
												</template>
											</el-table-column>
											<el-table-column label='<%=rb.getString("WangGuan")%>' prop="gateway" show-overflow-tooltip></el-table-column>
											<el-table-column label='<%=rb.getString("MuDiWangLuo")%>' prop="netAddr" show-overflow-tooltip></el-table-column>
											<el-table-column label='<%=rb.getString("ZiWangYanMa")%>' prop="netMask" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-form-item prop='staticRoute' style="display:none;" label="" label-width="0px">
											<el-input v-model='gsmImportAddOrEditForm.staticRoute'></el-input>
										</el-form-item>	
									</div>
								</div>
								<!--WAN config-->
								<div>
									<div class='commonFlex' style="padding: 0 0 0 0px; justify-content: space-between;">
										<div>
											<span class='tieleTipCircle'></span>
											<span class='commonText14'>WAN Config</span>		
											<span class='commonNotes12' style="margin-left: 10px"><%=rb.getString("GSMXianZhiTiaoShu12")%></span>
										</div>
										<div v-if="gsmOperType != 'view' && gsmImportAddOrEditForm.wan.length < 12" class='licenseImportBox' @click="importParamClick('', 'add', 'importWan', '')" style="margin: 0;">
											<i class='el-icon el-icon-plus importIcon'></i>
										</div>	
									</div> 
									<div style='padding: 5px 0px 20px 0px;' class='paramPoolWarp'>
										<el-ctable ref="importWanListTable" :rownumber="true" id="importWanListTable" :data="gsmImportAddOrEditForm.wan" height="200px" :pagination="false" class="enableShowOrHide" style="border:1px solid #E9E9E9;">
											<el-table-column label='<%=rb.getString("CaoZuo")%>' min-width="100">
												<template slot-scope="scope">
													<span class="el-icon el-icon-operation-edit" @click="importParamClick(scope.row, 'edit', 'importWan', '')" style="margin-right:15px;"></span>
													<span class="el-icon el-icon-operation-delete" @click="importParamDelClick(scope.row, 'importWan', '')" ></span>
												</template>
											</el-table-column>
											<el-table-column label='<%=rb.getString("SheZhiKaiGuan")%>' prop="enable" width='100px' show-overflow-tooltip>
												<template slot-scope="scope">
													<el-switch onclick="event.stopPropagation()" v-model="scope.row.enable" active-value="1" inactive-value="0"></el-switch>
												</template>
											</el-table-column>
											<el-table-column label='<%=rb.getString("gsmIPFangWenFangShi")%>' prop="ipMode" show-overflow-tooltip>
												<template slot-scope="scope">
													<div v-if="scope.row.ipMode == '0'">DHCP</div>
													<div v-if="scope.row.ipMode == '1'">Static</div>
												</template>
											</el-table-column>
											<el-table-column label='<%=rb.getString("IPDiZhi")%>' prop="ipAddr" show-overflow-tooltip></el-table-column>
											<el-table-column label='<%=rb.getString("ZiWangYanMa")%>' prop="netMask" show-overflow-tooltip></el-table-column>
											<el-table-column label='<%=rb.getString("WangGuan")%>' prop="gateway" show-overflow-tooltip></el-table-column>
											<el-table-column label='VLAN ID' prop="vlanId" show-overflow-tooltip></el-table-column>
										</el-ctable>
										<el-form-item prop='wan' style="display:none;" label="" label-width="0px">
											<el-input v-model='gsmImportAddOrEditForm.wan'></el-input>
										</el-form-item>	
									</div>
								</div>
							</div>
						</el-collapse-item>
					</el-collapse>
				</el-form>
			</div>
			<div slot="footer" class="importFooter">
				<el-button type="primary" @click="paramsImportEditDialogSubmit" :disabled='gsmParamImportSaveBtnDisabled'><%=rb.getString("QueDing")%></el-button>
				<el-button @click="paramsImportEditDialogClose"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</el-dialog>
		<!--batch import:修改 弹窗中的按钮操作： 新建，修改 第二层 弹框-->
		<el-dialog class="gsmImportConfigAddDialog" :title='batchImportDialogTitle' top="25vh" width="50%" :visible.sync="paramsImportSaveDialogShow" 
			@close="paramsImportSaveDialogClose" :close-on-click-modal="false" append-to-body>
			<div class="gsmConfigAddMainBoxCls">
				<el-form ref="gsmImportSaveForm" :model="gsmImportSaveForm" :rules="gsmImportSaveFormRules" label-position="top" style='min-height: 160px;'>

					<div v-if="gsmCommonImportOperModel == 'importRoute'" style='display:flex;margin-left:16px;flex-wrap: wrap'>
						
						<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop="onboot" class="titleAndNoteBox commonParamImportItem">
							<el-switch onclick="event.stopPropagation()" v-model="gsmImportSaveForm.onboot" active-value="1" inactive-value="0"></el-switch>
						</el-form-item>	
						<el-form-item label='<%=rb.getString("WangGuan")%>' prop='gateway' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gsmImportSaveForm.gateway'></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("MuDiWangLuo")%>' prop='netAddr' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gsmImportSaveForm.netAddr'></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("GNBZiWangYanMa")%>' prop='netMask' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gsmImportSaveForm.netMask'></el-input>
						</el-form-item>
					</div>

					<div v-if="gsmCommonImportOperModel == 'importWan'" style='display:flex;margin-left:16px;flex-wrap: wrap'>
						<el-form-item label='<%=rb.getString("SheZhiKaiGuan")%>' prop="enable" class="titleAndNoteBox " style="width: 100%;">
							<el-switch onclick="event.stopPropagation()" v-model="gsmImportSaveForm.enable" active-value="1" inactive-value="0"></el-switch>
						</el-form-item>	
						<el-form-item label='<%=rb.getString("gsmIPFangWenFangShi")%>' prop='ipMode' class="titleAndNoteBox commonParamImportItem">
							<el-select v-model='gsmImportSaveForm.ipMode'>
								<el-option label='DHCP' value='0'></el-option>
								<el-option label='Static' value='1'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item v-if="gsmImportSaveForm.ipMode == '1'" label='<%=rb.getString("IPDiZhi")%>' prop='ipAddr' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gsmImportSaveForm.ipAddr'></el-input>
						</el-form-item>
						<el-form-item v-if="gsmImportSaveForm.ipMode == '1'" label='<%=rb.getString("GNBZiWangYanMa")%>' prop='netMask' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gsmImportSaveForm.netMask'></el-input>
						</el-form-item>
						<el-form-item v-if="gsmImportSaveForm.ipMode == '1'" label='<%=rb.getString("WangGuan")%>' prop='gateway' class="titleAndNoteBox commonParamImportItem">
							<el-input v-model.trim='gsmImportSaveForm.gateway'></el-input>
						</el-form-item>
						<el-form-item label='VLAN ID' prop="vlanId" class='inputCommon validate-item commonParamImportItem'>
							<el-input v-model.trim="gsmImportSaveForm.vlanId">
								<template slot="append"><%=rb.getString("FanWei")%>: 1~4094</template>
							</el-input>
						</el-form-item>
					</div>
				</el-form>
			</div>
			<div slot="footer" class="importFooter">
				<el-button type="primary" @click="paramsImportSaveDialogSubmit"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="paramsImportSaveDialogClose"><%=rb.getString("QuXiao")%></el-button>
			</div>	
		</el-dialog>
		<!--batch import: view -->
		<div class='rightBox infoSpecifiedDevice infoImportBox' style='position: relative;' v-show='gsmCommonImportFileInfoShow'>
			<div class='commonFlex commonContent commonTitle rightHeaderBox' style='padding: 0 20px;'>
   				<span class='AddTitle commonText14' style='font-size: 16px;'>SN:{{importFileInfoSN}}</span>
   				<span class='closeIconBox' @click='commonImportFileInfoCancel'><i class='el-icon el-icon-close'></i></span>
   			</div>
   			<div class='rightContent' style='padding: 0 20px;'>
   				<div style='height: calc(100% - 50px);overflow-y: scroll;'>
   					<el-collapse v-model="infoActiveName">
                        <el-collapse-item name="basic">
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'><%=rb.getString("GSMJiChuXinXi")%></span></p></template>
                            <div class='commonBorderBottom'>
								<div class='commonText' style='padding-top: 0;'>
                                    <span class='commonTitleText12'>IpaUnitId</span>
                                    <span class='commonGeneral12 marginRight5'>{{gsmParamsImportInfo.ipaUnitid}}</span>
                                </div>
                                <div class='commonText' style='padding-top: 0;'>
                                    <span class='commonTitleText12'>OmlRemoteIp</span>
                                    <span class='commonGeneral12 marginRight5'>{{gsmParamsImportInfo.omlRemoteIp}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'>OmlRemoteIpBak</span>
                                    <span class='commonGeneral12 marginRight5'>{{gsmParamsImportInfo.omlRemoteIpBak}}</span>
                                </div>
                                <div class='commonText'>
                                    <span class='commonTitleText12'><%=rb.getString("GSMShePinGongLv")%></span>
                                    <span class='commonGeneral12 marginRight5'>{{gsmParamsImportInfo.rfPower}}</span>
                                </div>
                            </div>
                        </el-collapse-item>
                        <el-collapse-item name="route">
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'><%=rb.getString("GSMLuYou")%></span></p></template>
							<div class='commonBorderBottom'>
                                <div class='commonBorder' style='padding-top: 5px; margin-bottom: 5px;background: #FCFCFC;' v-for='item in gsmParamsImportInfo.staticRoute'>
									<div class='commonText1' style='padding: 6px 10px 3px;'>
										<span class='commonTitleText12'><%=rb.getString("SheZhiKaiGuan")%></span>
										<el-switch disabled v-model="item.onboot" active-value="1" inactive-value="0"></el-switch>
									</div>
									<div class='commonText1' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("WangGuan")%></span>
										<span class='commonGeneral12'>{{item.gateway}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'><%=rb.getString("MuDiWangLuo")%></span>
										<span class='commonGeneral12'>{{item.netAddr}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("ZiWangYanMa")%></span>
										<span class='commonGeneral12'>{{item.netMask}}</span>
									</div>
								</div>
                            </div>
                        </el-collapse-item>
						<el-collapse-item name="wanConfig">
                            <template slot='title'><p style="display:inline-block;margin-left: 26px;"><span class='commonGeneralBold12'>WAN Config</span></p></template>
							<div class='commonBorderBottom'>
                                <div class='commonBorder wanDataInfoCls' style='padding-top: 5px; margin-bottom: 5px;background: #FCFCFC;' v-for='item in gsmParamsImportInfo.wan'>
									<div class='commonText1 enableShowOrHide' style='padding: 6px 10px 3px;' >
										<span class='commonTitleText12'><%=rb.getString("SheZhiKaiGuan")%></span>
										<el-switch disabled v-model="item.enable" active-value="1" inactive-value="0"></el-switch>
									</div>
									<div class='commonText1' style='padding: 6px 10px 3px;'>
										<span class='commonTitleText12'><%=rb.getString("gsmIPFangWenFangShi")%></span>
										<span class='commonGeneral12' v-if='item.ipMode == "0"'>DHCP</span>
										<span class='commonGeneral12' v-if='item.ipMode == "1"'>Static</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("IPDiZhi")%></span>
										<span class='commonGeneral12'>{{item.ipAddr}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'><%=rb.getString("GNBZiWangYanMa")%></span>
										<span class='commonGeneral12'>{{item.netMask}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px;'>
										<span class='commonTitleText12'><%=rb.getString("WangGuan")%></span>
										<span class='commonGeneral12'>{{item.gateway}}</span>
									</div>
									<div class='commonText1' style='padding: 3px 10px 6px;'>
										<span class='commonTitleText12'>VLAN ID</span>
										<span class='commonGeneral12'>{{item.vlanId}}</span>
									</div>
								</div>
                            </div>
                        </el-collapse-item>
   					</el-collapse>
				</div>
			</div>
		</div>
	</div>
</div>
<script>
    var gsmAddOrEditConfigVue = new Vue({
        el: '#gsmAddOrEditConfigPage',
        data() {
            var vm = this,
	            validatePolicyName = function(rule,value,callback){//basic info：policy name
					if(value === '' || value === null || value === undefined) {
						callback(new Error('<%=rb.getString("BiTian")%>'));
					}else{
						callback();
					}
				},
				validateVersion = function(rule, value, callback) {//upgrade: original version
					var verType = vm.specifyVersionType;
					if(vm.gsmAddOrEditForm.upgradeEnable == '1' && vm.gsmOperType != 'view'){
	                    if(verType == '1') {
	                    	callback();
	                    }else if(vm.gsmResultOriginalVersionList.length == 0) {
	                    	callback('<%=rb.getString("QingXuanZeJiLu") %>');
	                    }else{
	                    	callback();
	                    }
                    }else{
                    	callback();
                    }
                },
                validateTarget = function(rule,value,callback) {//upgrade: target version
					if(vm.gsmAddOrEditForm.upgradeEnable == '1' && vm.gsmOperType != 'view'){
	                	if( value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("BiTian")%>');
						}else {
							callback();
						}
					}else{
                    	callback();
                    }
				},
				validateFilesName = function(rule,value,callback) {//license import: file name
	            	var value = vm.fileName;
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else {
						callback();
					}
				},
				validateFileNames = (rule,value,callback) => {//batch import: file name
					var value = vm.fileName; 
					if( value === '' || value === null || value === undefined) {
						callback('<%=rb.getString("QingXianXuanZeWenJian")%>');
					}else if(!fileFormatMatch(value,"xlsx,xls")){
						callback(new Error('<%=rb.getString("DangQianZhiChiWenJianLeiXing")%>'));
					}else {
						callback();
					} 
				},
				//<%=rb.getString("FanWei")%>:Ipa[0~65535],Id[0~255]
				validateIpaUnitid = function(rule,value,callback) {
					//value xx-xx 形式， -前范围0-65535，-后范围0-255
					var reg = /^([0-9]{1,5})-([0-9]{1,3})$/;
					if(rule.type == 'common'){
						if(vm.gsmAddOrEditForm.selfConfigEnable == '1' && vm.gsmAddOrEditForm.paramConfigEnable == '1' && vm.gsmOperType != 'view'){
							if( value === '' || value === null || value === undefined) {
								callback('<%=rb.getString("BiTian")%>');
							}else{
								if(reg.test(value)){
									var arr = value.split('-');
									if(arr[0] < 0 || arr[0] > 65535 || arr[1] < 0 || arr[1] > 255){
										callback(new Error('<%=rb.getString("FanWei")%>: Ipa[0~65535],Id[0~255]'))
									}else{
										callback();
									}
								}else{
									callback(new Error('<%=rb.getString("FanWei")%>: Ipa[0~65535],Id[0~255]'))
								}
							}
						}else{
							if(reg.test(value)){
								var arr = value.split('-');
								if(arr[0] < 0 || arr[0] > 65535 || arr[1] < 0 || arr[1] > 255){
									callback(new Error('<%=rb.getString("FanWei")%>: Ipa[0~65535],Id[0~255]'))
								}else{
									callback();
								}
							}else{
								callback(new Error('<%=rb.getString("FanWei")%>: Ipa[0~65535],Id[0~255]'))
							}
						}
					}else{
						if(value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("BiTian")%>');
						}else{
							if(reg.test(value)){
								var arr = value.split('-');
								if(arr[0] < 0 || arr[0] > 65535 || arr[1] < 0 || arr[1] > 255){
									callback(new Error('<%=rb.getString("FanWei")%>: Ipa[0~65535],Id[0~255]'))
								}else{
									callback();
								}
							}else{
								callback(new Error('<%=rb.getString("FanWei")%>: Ipa[0~65535],Id[0~255]'))
							}
						}
					}
				},
				validateIp = function(rule,value,callback) {
					if(rule.type == 'common'){
						if(vm.gsmAddOrEditForm.selfConfigEnable == '1' && vm.gsmAddOrEditForm.paramConfigEnable == '1' && vm.gsmOperType != 'view'){
							if( value === '' || value === null || value === undefined) {
								callback('<%=rb.getString("BiTian")%>');
							}else if(vm.isValidIP(value)){
								callback();
							}else{
								callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
							}
						}else{
							if(vm.isValidIP(value)){
								callback();
							}else{
								callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
							}
						}
					}else{
						if( value === '' || value === null || value === undefined) {
							callback('<%=rb.getString("BiTian")%>');
						}else if(vm.isValidIP(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}
				},
				validateIpBak = function(rule,value,callback) {
					if( value === '' || value === null || value === undefined) {
						callback();
					}else{
						if(vm.isValidIP(value)){
							callback();
						}else{
							callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
						}
					}
				},
				validateRfPower= function(rule,value,callback) {
					if( value === '' || value === null || value === undefined) {
						callback();
					}else{
						if(vm.isNumeric(value)&&parseInt(value)>=24 && parseInt(value)<=43){
							callback();
						}else{
							callback(new Error('<%=rb.getString("FanWei")%>: 24~43'))
						}
					}
				},
				validateGateway = (rule,value,callback) => {
					if(value == '' || value == undefined || value == null){
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}else if(vm.isValidIP(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}	
				},
				validateNetmask = (rule,value,callback) => {
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
					}else if(vm.isMask(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuHeFaDeYanMa")%>'))
					}
				},
				validateNetAddr = (rule,value,callback) => {
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}else if(vm.isValidIP(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}
				},
				validateImportWanIpAddress = (rule,value,callback) => {
					if(value === '' || value === null || value === undefined){
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}else if(vm.isValidIP(value)){
						callback();
					}else{
						callback(new Error('<%=rb.getString("QingShuRuYiGeYouXiaoDeIPDiZhi")%>'))
					}
				},
				validateWanVlanID = (rule,value,callback) => {
					if(value === '' || value === null || value === undefined){
						callback();
					}else if(vm.isNumeric(value)&&parseInt(value)>=1 && parseInt(value)<=4094){
						callback();
					}else{
						callback(new Error('<%=rb.getString("FanWei")%>：1~4094,<%=rb.getString("ZhengXing")%>'))
					}			
				}
            return {
				gsmPolicyTitle: 'Add Policy',
				height:"100%",
				gsmOperType: '',
				gsmPolicyId: '',
				commonImportRow: {},				
				gsmAddOrEditForm: {
            		policySwitch: '1', //policy enable
            		policyName: '',
            		productType: '',
            		executeType: '0',
					upgradeEnable: '0', //upgrade enable
            		targetVersion: '',
            		originalVersion: '',
            		preserveSetting: '1',
					licenseEnable: '0', // license enable
					selfConfigEnable: '0', //parameters config enable
            		paramConfigEnable: '0', //common params enable
					ipaUnitid: '0-0',
					omlRemoteIp: '0.0.0.0',
					omlRemoteIpBak: '', 
					rfPower: '',

					advanceEnable: '0', //common params advance enable
					dns1: '',
					dns2: '',
					localTimezoneName: '',
					staticRoute: [],
					wan: [],
					custParam: [],
					batchConfigEnable: '0', //batch import enable
				},
				gsmAddOrEditRule: {
					policyName: [{validator: validatePolicyName}],
					productType: [{validator: validatePolicyName}],
					originalVersion: [{validator: validateVersion}],
					targetVersion: [{validator: validateTarget}],

					ipaUnitid: [{required: true, type: 'common', validator: validateIpaUnitid}],
					omlRemoteIp: [{required: true, type: 'common', validator: validateIp}],
					omlRemoteIpBak: [{validator: validateIpBak}],
					rfPower: [{validator: validateRfPower}],
					dns1: [{validator: validateIpBak}],
					dns2: [{validator: validateIpBak}],
				},
				gsmProductList: [],
				gsmFunctionModulesSelect: '0',
				specifyVersionType: '0', //any checkbox
				gsmAddVersionBtnShow: true,
				gsmResultOriginalVersionShow: false,
				gsmSearchValue: "",
				//upgrade
				gsmTarVersionList: [], //software upgrade
                gsmAddVersionShow: false,
                gsmAddVersionBtnShow: true,
                gsmResultOriginalVersionShow: false,
            	gsmSoftwareAddVersionForm: {
            		versionStr: '',
					versionList: [],
					itemTest:'',
					originalVersion: '',
            	},
            	gsmVersionErrorMessage: '',
            	gsmVersionMessage: '',
            	gsmResultOriginalVersionList: [],
            	gsmSoftwareOriginalVersionList: [],
				gsmCurSelectVersionData: [],
				//license
				gsmLicenseUrl: '${ctx}/gsm/pnp/queryGsmPnPLicensePageList.action',
                gsmLicenseQuery: {
					searchText: '',
					productType: '',
					timeZone: timeZone
				},
				gsmCommonImportShow: false,
				commonImportType: '',
				commonImportTitle: '',
				gsmImportModeSelection:'moreFile',
	            //fileName:'',
	            gsmLicenseForm: {uploadFileUrl: ''},
	            gsmMoreFileParams:{},
	            gsmMoreFileList:[],
	            gsmFileData:[],
				gsmImportLicenseRules: {fileName:[{validator: validateFilesName}]},
	            gsmCommonUploadBtnDisabled: false, //license import or batch import submit button disabled
				gsmLicenseLoading: false,
				//common params
				gsmActiveName: ['basic', 'advance', 'customized'],
				timeZoneList: [],
				gsmCommonAddOrUpdateParamShow: false,
				gsmCommonAddOrUpdateParamTitle: '<%=rb.getString("TianJia")%>',
				gsmCommonAddEditForm: {
					onboot: '1',
					gateway: '', //公共
					netAddr: '',//router //Destination Network ip
					netMask: '', //公共 255.255.255.0

					enable: '1',
					ipAddr: '',
					ipMode: '0',//wan
					vlanId: '',
				},
				gsmCommonAddEditFormRules: {
					gateway:[{required: true, validator: validateGateway, trigger:'blur'}],	
					netMask:[{required: true, validator: validateNetmask, trigger:'blur'}],	
					netAddr:[{required: true, validator: validateNetAddr, trigger:'blur'}],
					ipAddr:[{required: true, validator: validateImportWanIpAddress, trigger:'blur'}],
					vlanId:[{ validator: validateWanVlanID, trigger:'blur'}],
				},
				gsmCommonImportOperType: '',
				gsmCommonImportOperModel: '',
				gsmCommonAdvanceOperType:'',
				gsmCommonAdvanceOperModel:'',
				commonAddOrUpdateBtnDisabled: false,
				//------------------------------批量导入 详情+ 修改
				//edit
				paramsImportEditDialogShow: false,
				//info
				gsmCommonImportFileInfoShow: false,
				infoActiveName: ['basic','route','wanConfig'],
				paramsImportSaveDialogShow: false,
				gsmImportActiveName: ['basic', 'advance'],
				gsmImportSaveForm: {
					onboot: '1',
					gateway: '', //公共
					netAddr: '',//router
					netMask: '', //公共

					enable: '1',
					ipAddr: '',
					ipMode: '0',//wan
					vlanId: '' //公共
				},
				gsmImportSaveFormRules: {
					gateway:[{required: true, validator: validateGateway, trigger:'blur'}],	
					netMask:[{required: true, validator: validateNetmask, trigger:'blur'}],	
					netAddr:[{required: true, validator: validateNetAddr, trigger:'blur'}],
					ipAddr:[{required: true, validator: validateImportWanIpAddress, trigger:'blur'}],
					vlanId:[{ validator: validateWanVlanID, trigger:'blur'}],
				},
				gsmImportAddOrEditForm: {
					ipaUnitid: '0-0',
					omlRemoteIp: '0.0.0.0',
					omlRemoteIpBak: '',
					rfPower: '',
					//advanceEnable: '1',// ？？？ 原型有，接口参数没
					staticRoute: [],
					wan: []
				},
				gsmImportAddOrEditRules: {
					ipaUnitid: [{required: true, type: 'import', validator: validateIpaUnitid}],
					omlRemoteIp: [{required: true, type: 'import', validator: validateIp}],
					omlRemoteIpBak: [{validator: validateIpBak}],
					rfPower: [{validator: validateRfPower}]
				},
				//batch import
				gsmParameterConfigActive: 'gsmParamsConfig',

				gsmUrlConfigPlan:"",
				importParamsQuery:{
					timeZone: timeZone,
					searchText: ""
				},
				gsmImportRuleForm: {
					uploadFileUrl: '',
					operType: "0",
				},
				gsmImportRules:{ fileName:[{ validator: validateFileNames}]},	         	          
				fileParams:{},              
				fileName:'', // 注意这里的逻辑，fileName 用于显示，fileParams 用于上传	            					
				fileList:[],
				filePath:'',
				commonImportFileLoading: false,
				downloadVisible: false, // batch import failed dialog
				gsmParamImportSaveBtnDisabled: false,
				importFileInfoSN: '',
				gsmParamsImportInfo:{
					ipaUnitid: '',
					omlRemoteIp: '',
					omlRemoteIpBak: '', 
					rfPower: '',
					//advanceEnable: '',
					staticRoute: [],
					wan: [],
				},

				gsmSaveBtnDisabled: false,
            }
        },
        computed: {
			batchImportDialogTitle(){
				var vm = this;
				return vm.gsmCommonImportOperType == 'add' ? '<%=rb.getString("TianJia")%>' : '<%=rb.getString("XiuGai")%>'
			},
        },
        watch: {
            'gsmAddOrEditForm.productType': function(val) {
                var vm = this,  
					params = {  product_value: val };
                axios.post("${ctx}/plugandplay/policy/getTargetVersion.action", stringify(params)).then(function(response){ // get target version
					var data = response.data;
					if(data.length > 0){
						vm.gsmTarVersionList = data;
					}
				});
                vm.gsmLicenseQuery.productType = val;//product change，license query 
            },
			// any checke: add version dialog hidden
			specifyVersionType: function(val){
				var vm = this;
				if(val == '1'){
					vm.gsmAddVersionShow = false;
				}
			}
        },
        methods: {
			//init
			gsmPolicyInit(type, row){
				var vm = this,
					timeStr = "Africa/Abidjan,Africa/Accra,Africa/Addis_Ababa,Africa/Algiers,Africa/Asmara,Africa/Bamako,Africa/Bangui,Africa/Banjul,Africa/Bissau,Africa/Blantyre,Africa/Brazzaville,Africa/Bujumbura,Africa/Cairo,Africa/Casablanca,Africa/Ceuta,Africa/Conakry,Africa/Dakar,Africa/Dar_es_Salaam,Africa/Djibouti,Africa/Douala,Africa/El_Aaiun,Africa/Freetown,Africa/Gaborone,Africa/Harare,Africa/Johannesburg,Africa/Juba,Africa/Kampala,Africa/Khartoum,Africa/Kigali,Africa/Kinshasa,Africa/Lagos,Africa/Libreville,Africa/Lome,Africa/Luanda,Africa/Lubumbashi,Africa/Lusaka,Africa/Malabo,Africa/Maputo,Africa/Maseru,Africa/Mbabane,Africa/Mogadishu,Africa/Monrovia,Africa/Nairobi,Africa/Ndjamena,Africa/Niamey,Africa/Nouakchott,Africa/Ouagadougou,Africa/Porto-Novo,Africa/Sao_Tome,Africa/Tripoli,Africa/Tunis,Africa/Windhoek,America/Adak,America/Anchorage,America/Anguilla,America/Antigua,America/Araguaina,America/Argentina/Buenos_Aires,America/Argentina/Catamarca,America/Argentina/Cordoba,America/Argentina/Jujuy,America/Argentina/La_Rioja,America/Argentina/Mendoza,America/Argentina/Rio_Gallegos,America/Argentina/Salta,America/Argentina/San_Juan,America/Argentina/San_Luis,America/Argentina/Tucuman,America/Argentina/Ushuaia,America/Aruba,America/Asuncion,America/Atikokan,America/Bahia,America/Bahia_Banderas,America/Barbados,America/Belem,America/Belize,America/Blanc-Sablon,America/Boa_Vista,America/Bogota,America/Boise,America/Cambridge_Bay,America/Campo_Grande,America/Cancun,America/Caracas,America/Cayenne,America/Cayman,America/Chicago,America/Chihuahua,America/Costa_Rica,America/Creston,America/Cuiaba,America/Curacao,America/Danmarkshavn,America/Dawson,America/Dawson_Creek,America/Denver,America/Detroit,America/Dominica,America/Edmonton,America/Eirunepe,America/El_Salvador,America/Fortaleza,America/Glace_Bay,America/Godthab,America/Goose_Bay,America/Grand_Turk,America/Grenada,America/Guadeloupe,America/Guatemala,America/Guayaquil,America/Guyana,America/Halifax,America/Havana,America/Hermosillo,America/Indiana/Indianapolis,America/Indiana/Knox,America/Indiana/Marengo,America/Indiana/Petersburg,America/Indiana/Tell_City,America/Indiana/Vevay,America/Indiana/Vincennes,America/Indiana/Winamac,America/Inuvik,America/Iqaluit,America/Jamaica,America/Juneau,America/Kentucky/Louisville,America/Kentucky/Monticello,America/Kralendijk,America/La_Paz,America/Lima,America/Los_Angeles,America/Lower_Princes,America/Maceio,America/Managua,America/Manaus,America/Marigot,America/Martinique,America/Matamoros,America/Mazatlan,America/Menominee,America/Merida,America/Metlakatla,America/Mexico_City,America/Miquelon,America/Moncton,America/Monterrey,America/Montevideo,America/Montserrat,America/Nassau,America/New_York,America/Nipigon,America/Nome,America/Noronha,America/North_Dakota/Beulah,America/North_Dakota/Center,America/North_Dakota/New_Salem,America/Ojinaga,America/Panama,America/Pangnirtung,America/Paramaribo,America/Phoenix,America/Port_of_Spain,America/Port-au-Prince,America/Porto_Velho,America/Puerto_Rico,America/Rainy_River,America/Rankin_Inlet,America/Recife,America/Regina,America/Resolute,America/Rio_Branco,America/Santa_Isabel,America/Santarem,America/Santiago,America/Santo_Domingo,America/Sao_Paulo,America/Scoresbysund,America/Sitka,America/St_Barthelemy,America/St_Johns,America/St_Kitts,America/St_Lucia,America/St_Thomas,America/St_Vincent,America/Swift_Current,America/Tegucigalpa,America/Thule,America/Thunder_Bay,America/Tijuana,America/Toronto,America/Tortola,America/Vancouver,America/Whitehorse,America/Winnipeg,America/Yakutat,America/Yellowknife,Antarctica/Casey,Antarctica/Davis,Antarctica/DumontDUrville,Antarctica/Macquarie,Antarctica/Mawson,Antarctica/McMurdo,Antarctica/Palmer,Antarctica/Rothera,Antarctica/Syowa,Antarctica/Troll,Antarctica/Vostok,Arctic/Longyearbyen,Asia/Aden,Asia/Almaty,Asia/Amman,Asia/Anadyr,Asia/Aqtau,Asia/Aqtobe,Asia/Ashgabat,Asia/Baghdad,Asia/Bahrain,Asia/Baku,Asia/Bangkok,Asia/Beirut,Asia/Bishkek,Asia/Brunei,Asia/Chita,Asia/Choibalsan,Asia/Colombo,Asia/Damascus,Asia/Dhaka,Asia/Dili,Asia/Dubai,Asia/Dushanbe,Asia/Gaza,Asia/Hebron,Asia/Ho_Chi_Minh,Asia/Hong_Kong,Asia/Hovd,Asia/Irkutsk,Asia/Jakarta,Asia/Jayapura,Asia/Jerusalem,Asia/Kabul,Asia/Kamchatka,Asia/Karachi,Asia/Kathmandu,Asia/Khandyga,Asia/Kolkata,Asia/Krasnoyarsk,Asia/Kuala_Lumpur,Asia/Kuching,Asia/Kuwait,Asia/Macau,Asia/Magadan,Asia/Makassar,Asia/Manila,Asia/Muscat,Asia/Nicosia,Asia/Novokuznetsk,Asia/Novosibirsk,Asia/Omsk,Asia/Oral,Asia/Phnom_Penh,Asia/Pontianak,Asia/Pyongyang,Asia/Qatar,Asia/Qyzylorda,Asia/Rangoon,Asia/Riyadh,Asia/Sakhalin,Asia/Samarkand,Asia/Seoul,Asia/Shanghai,Asia/Singapore,Asia/Srednekolymsk,Asia/Taipei,Asia/Tashkent,Asia/Tbilisi,Asia/Thimphu,Asia/Tokyo,Asia/Ulaanbaatar,Asia/Urumqi,Asia/Ust-Nera,Asia/Vientiane,Asia/Vladivostok,Asia/Yakutsk,Asia/Yekaterinburg,Asia/Yerevan,Atlantic/Azores,Atlantic/Bermuda,Atlantic/Canary,Atlantic/Cape_Verde,Atlantic/Faroe,Atlantic/Madeira,Atlantic/Reykjavik,Atlantic/South_Georgia,Atlantic/St_Helena,Atlantic/Stanley,Australia/Adelaide,Australia/Brisbane,Australia/Broken_Hill,Australia/Currie,Australia/Darwin,Australia/Eucla,Australia/Hobart,Australia/Lindeman,Australia/Lord_Howe,Australia/Melbourne,Australia/Perth,Australia/Sydney,Europe/Amsterdam,Europe/Andorra,Europe/Athens,Europe/Belgrade,Europe/Berlin,Europe/Bratislava,Europe/Brussels,Europe/Bucharest,Europe/Budapest,Europe/Busingen,Europe/Chisinau,Europe/Copenhagen,Europe/Dublin,Europe/Gibraltar,Europe/Guernsey,Europe/Helsinki,Europe/Isle_of_Man,Europe/Istanbul,Europe/Jersey,Europe/Kaliningrad,Europe/Kiev,Europe/Lisbon,Europe/Ljubljana,Europe/London,Europe/Luxembourg,Europe/Madrid,Europe/Malta,Europe/Mariehamn,Europe/Minsk,Europe/Monaco,Europe/Moscow,Europe/Oslo,Europe/Paris,Europe/Podgorica,Europe/Prague,Europe/Riga,Europe/Rome,Europe/Samara,Europe/San_Marino,Europe/Sarajevo,Europe/Simferopol,Europe/Skopje,Europe/Sofia,Europe/Stockholm,Europe/Tallinn,Europe/Tirane,Europe/Uzhgorod,Europe/Vaduz,Europe/Vatican,Europe/Vienna,Europe/Vilnius,Europe/Volgograd,Europe/Warsaw,Europe/Zagreb,Europe/Zaporozhye,Europe/Zurich,Indian/Antananarivo,Indian/Chagos,Indian/Christmas,Indian/Cocos,Indian/Comoro,Indian/Kerguelen,Indian/Mahe,Indian/Maldives,Indian/Mauritius,Indian/Mayotte,Indian/Reunion,Pacific/Apia,Pacific/Auckland,Pacific/Bougainville,Pacific/Chatham,Pacific/Chuuk,Pacific/Easter,Pacific/Efate,Pacific/Enderbury,Pacific/Fakaofo,Pacific/Fiji,Pacific/Funafuti,Pacific/Galapagos,Pacific/Gambier,Pacific/Guadalcanal,Pacific/Guam,Pacific/Honolulu,Pacific/Johnston,Pacific/Kiritimati,Pacific/Kosrae,Pacific/Kwajalein,Pacific/Majuro,Pacific/Marquesas,Pacific/Midway,Pacific/Nauru,Pacific/Niue,Pacific/Norfolk,Pacific/Noumea,Pacific/Pago_Pago,Pacific/Palau,Pacific/Pitcairn,Pacific/Pohnpei,Pacific/Port_Moresby,Pacific/Rarotonga,Pacific/Saipan,Pacific/Tahiti,Pacific/Tarawa,Pacific/Tongatapu,Pacific/Wake,Pacific/Wallis",
					timeList = timeStr.split(",");

					vm.gsmOperType = type;
				if(type == 'add'){
                	vm.gsmPolicyTitle = '<%=rb.getString("XinZengCeLue")%>';
					vm.importParamsQuery.policyId = '${policyId}';
                }else if(type == 'modify'){
                	vm.gsmPolicyTitle = '<%=rb.getString("XiuGaiCeLue")%>';
					vm.importParamsQuery.policyId = row.policyId;
                }else{
                	vm.gsmPolicyTitle = '<%=rb.getString("ChaKanCeLue")%>';
					vm.importParamsQuery.policyId = row.policyId;
                }
				//获取时区
				vm.timeZoneList = timeList.map(item=>{
					return { label: item, value: item }
				})
				// get Product Type
				axios.post("${ctx}/gsm/pnp/getProductTypeSelect.action").then(function(response){ 
                	var data = response.data;
                	if(data.length > 0){
                		vm.gsmProductList = data;
    					if(type == 'add'){
    						vm.gsmAddOrEditForm.productType = vm.gsmProductList[0].value;
    					}else{
							//info or modify，get policy info
							vm.gsmPolicyId = row.policyId;
							vm.gsmAddOrEditForm.policySwitch = row.switch == '1' ? '1' : '0';
							vm.gsmAddOrEditForm.selfConfigEnable = row.configEnable == '1' ? '1' : '0';	

							Object.assign(vm.gsmAddOrEditForm, row);
							
							vm.gsmGetParamsInfo(row.policyId);
							vm.$nextTick(function(){
								initForm(vm.$refs.gsmAddOrEditForm);
							})
						}
                	}
				});
			},
			// get upgrade info 
			gsmGetParamsInfo(policyId) {
	            var vm = this, 
					params = { policyId: policyId };
	            axios.post('${ctx}/gsm/pnp/getPnPUpgradePolicyInfo.action', stringify(params)).then(function(response){
					var data = response.data;
					if(data){
						vm.gsmAddOrEditForm.preserveSetting = data.preserveSetting; 
						vm.gsmAddOrEditForm.targetVersion = data.destVersion;

						if(data.originalVersion){
							if(data.originalVersion == 'all'){
								vm.specifyVersionType = '1';
							}else{
								var resultOriginlist = data.originalVersion.split(',');
								vm.gsmResultOriginalVersionList = resultOriginlist.map(function(item){ 
									return { originalVersion: item };
								});
							}
						}
						vm.gsmAddVersionBtnShow = false;
						vm.gsmResultOriginalVersionShow = true;
					}					
				});

				var commonParams = { policyId: policyId, serialNumber: 'default'}; // 批量配置传递 sn, 公共配置传递default
				//查询所有参数接口
				axios.post('${ctx}/gsm/pnp/getPnPPolicyConfigInfo.action', stringify(commonParams)).then(function(res){
					var data = res.data;
					
					if(data){
						Object.assign(vm.gsmAddOrEditForm, data);
					}					
				});
	        },
			//product change
			gsmProductTypeChange(type) { 
				var vm = this;
				vm.gsmAddOrEditForm.originalVersion = '';
				vm.gsmResultOriginalVersionList = [];
				vm.gsmAddOrEditForm.targetVersion = '';
				vm.gsmTarVersionList = [];
				vm.gsmCommonOriginalVerList();
			},
			//software upgrade,license,parameter config change, right model close
        	gsmSelectMethodChange(val){
        		var vm = this;
        		vm.gsmCommonRightModelClose();
        	},
			//right model close
			gsmCommonRightModelClose(){
				var vm = this;
				vm.gsmAddVersionShow = false; //version 
				vm.gsmCommonImportShow = false; //license or batch import
				vm.gsmCommonAddOrUpdateParamShow = false; //batch import edit
				vm.gsmCommonImportFileInfoShow = false; //batch import view
			},
			//add version icon click
			gsmAddVersionClick(){
				var vm = this;
				vm.gsmAddVersionShow = true;  
				vm.gsmCommonImportShow = false; 
				vm.gsmCommonAddOrUpdateParamShow = false; //batch import edit
				vm.gsmCommonImportFileInfoShow = false; //batch import view
				vm.gsmCurSelectVersionData = [];
				vm.gsmSoftwareAddVersionForm.versionList = [];
				vm.$refs.gsmSoftwareOriginalVersionTable.clearSelection();
				vm.gsmCommonOriginalVerList();
			},
			// initial version data
			gsmCommonOriginalVerList(){
				var vm = this;
				axios.post("${ctx}/gsm/pnp/queryOriginalUpgradeVersionPageList.action",stringify({
                	productType: vm.gsmAddOrEditForm.productType,
                    searchText: '',
                    page: 1,
                    rows: 50
                })).then(function(response){
					var data = response.data;
					if(data && data.rows.length > 0){
						vm.gsmSoftwareOriginalVersionList = data.rows;
					}
				});
			},
			//manually add version
			gsmAddVersionBtn(){
				var vm = this, 
					versionText = vm.gsmSoftwareAddVersionForm.versionStr, 
					str = '';
				if(versionText == ''){
					vm.gsmVersionErrorMessage = '<%=rb.getString("ShuRuBanBenHao")%>';
				}else{
					if(versionText){
						str = versionText;
						if(vm.gsmSoftwareAddVersionForm.versionList.indexOf(str) == -1){
							vm.gsmSoftwareAddVersionForm.versionList.push(str);
							vm.gsmSoftwareAddVersionForm.versionStr = '';
							vm.gsmVersionErrorMessage = '';
							vm.gsmVersionMessage = '';
							vm.$refs.gsmSoftwareAddVersionForm.validateField('itemTest');
						}else{
							vm.gsmVersionErrorMessage = '<%=rb.getString("YiCunZai")%>';
						}
					}else{
						vm.gsmVersionErrorMessage = '<%=rb.getString("ShuRuBanBenHao")%>';
					}
				}
			},
			//manually added versions: delete
			gsmDeleteVersion(item){
				var vm = this, 
					index = vm.gsmSoftwareAddVersionForm.versionList.indexOf(item);
				if(index !== -1){
					vm.gsmSoftwareAddVersionForm.versionList.splice(index,1)
				}
				vm.gsmVersionErrorMessage = '';
			},
			//Original Version List：selected
			gsmVersionBatchSelect(selection){
				var vm = this;
				vm.gsmCurSelectVersionData = selection;
			},
			//add version: submit
        	gsmAddVersionSubmit(){
        		var vm = this, 
 					manualVersionList = vm.gsmSoftwareAddVersionForm.versionList, // manually added versions
					versionList = vm.gsmCurSelectVersionData.map(function(item){ //Original Version List: selected
						return item.originalVersion
					});
				if(manualVersionList.length == 0 && vm.gsmCurSelectVersionData.length == 0){
					vm.gsmVersionMessage = '<%=rb.getString("ZhiShaoXuanZeYiZhongBanBenFangShi")%>';
        			return;
        		}else{
        			vm.gsmVersionMessage = '';
					//合并去重
					var mergeVersionList = manualVersionList.concat(versionList).filter(function(item,index,self){
						return self.indexOf(item) === index;
					});
					var curList = mergeVersionList.map(function(item){
						return {originalVersion: item }
					});
            		if(vm.gsmResultOriginalVersionList.length > 0){
						// 合并去重
						vm.gsmResultOriginalVersionList = vm.gsmResultOriginalVersionList.concat(curList).filter(function(item,index,self){
							return self.findIndex(function(item2){
								return item2.originalVersion == item.originalVersion;
							}) === index;
						});
            		}else{
						vm.gsmResultOriginalVersionList = curList;
            		}					
            		vm.gsmAddVersionShow = false;
            		vm.gsmAddVersionBtnShow = false;
                    vm.gsmResultOriginalVersionShow = true;
        		}
        	},
			//add version: cancel
        	gsmAddVersionCancel(){
        		var vm = this;
        		vm.gsmAddVersionShow = false;
        		vm.gsmCurSelectVersionData = [];
        		vm.$refs.gsmSoftwareAddVersionForm.resetFields();
        		vm.gsmSoftwareAddVersionForm.versionList = [];
        		vm.$refs.gsmSoftwareOriginalVersionTable.clearSelection();
        	},
			//Original Version: delete
        	gsmDeleteVersionItem(item){
				var vm = this, 
					index = vm.gsmResultOriginalVersionList.indexOf(item);
				if(index !== -1){
					vm.gsmResultOriginalVersionList.splice(index,1)
				}
			},
			//Original Version: clear 
			gsmClearVersionBtnClick(){
				var vm = this;
				vm.gsmResultOriginalVersionList = [];
			},
			//------------------------------------------------------------license------------------------------------------------------------
			//delete
			gsmDeleteLicenseClick(row,evt){
				var vm = this, 
					params = { fileNames : row.file_name };
				vm.gsmCommonImportShow = false;
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
	    						message: '<%=rb.getString("ChengGong")%>',
	    						type:'success'
	    					})
	    					vm.$refs.licenseTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch()
			},
			//import
			gsmImportLicenseClick(){
        		var vm = this;
				vm.commonImportTitle = '<%=rb.getString("DaoRu")%> License';
				vm.commonImportType = 'license';
				vm.gsmCommonImportShow = true; //license or batch import
				vm.gsmAddVersionShow = false; //version
				vm.gsmCommonAddOrUpdateParamShow = false; //batch import edit
				vm.gsmCommonImportFileInfoShow = false; //batch import view
			},
			gsmMoreCheckFile(res,file){ 
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
					vm.gsmCommonImportShow = false;
					vm.$refs.licenseTable.refresh();
					vm.gsmMoreCloseFileSelect();
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				var fileList = vm.$refs.moreUpload.uploadFile;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			//select file
			gsmMoreFileChange(file,fileList){
				var vm = this, 
					fileIndex = file.name.lastIndexOf("."), 
					fileType = file.name.substr(fileIndex + 1, file.name.length);
                if(['lic'].indexOf(fileType.toLowerCase()) === -1){
                    return false;
                }else{
					vm.fileName += vm.fileName ? ',' + file.name : file.name;
                    vm.gsmMoreFileParams.FileName = file.name;
                }
			},
			gsmMoreFileSelect(){
				var vm =this;
	    	    vm.fileName = '';
				vm.$refs.moreUpload.clearFiles();
				vm.$refs['more_file_up'].click();
			},
			gsmMoreCloseFileSelect(){
				var vm = this;
				vm.fileName = '';
				vm.$refs.moreUpload.clearFiles();
			},
			gsmMoreFileRequest(file){
				var vm = this;
				vm.gsmFileData.push(file.file);
			},
			//license or batch import submit
	        gsmCommonImportSubmit() {
				var vm = this, 
					fileImportArr = [], 
					productType = vm.gsmAddOrEditForm.productType;
				if(vm.commonImportType == 'license'){
					vm.$refs.gsmLicenseForm.validate((valid) => {
						if (valid){
							var fd = new FormData(),
								config = {
									headers: { 'Content-Type': 'multipart/form-data' }
								};
							vm.$refs.moreUpload.submit();
							vm.gsmCommonUploadBtnDisabled = true;
							vm.gsmLicenseLoading = true;
							if(vm.gsmFileData.length > 0){
								vm.gsmFileData.forEach(file =>{
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
							fd.append('fileArr',JSON.stringify(fileImportArr)); 
							fd.append('productType',productType);
							axios.post("${ctx}/cell/license/uploadLicenseFile.action",fd,config).then(function(res){
								if(res.data["success"]){
									vm.$message.success('<%=rb.getString("DaoRuChengGong")%>');
									vm.$refs.licenseTable.refresh();
								}else{
									vm.$message.error(res.data["message"]);
								}
								vm.gsmCommonImportCancel();
							})
						}
					})
				}else if(vm.commonImportType == 'batchParams'){
					vm.$refs.gsmImportRuleForm.validate((valid) => {
						if(valid){
							vm.$refs.upload.submit();
							vm.commonImportFileLoading = true;
							vm.gsmCommonUploadBtnDisabled = true;
						}
					})
				}
			},
			//close import right model
			gsmCommonImportCancel(){ 
				var vm = this;
				if(vm.commonImportType == 'license'){
					vm.gsmCommonImportShow = false;
					vm.gsmCommonUploadBtnDisabled = false;
					vm.gsmLicenseLoading = false;
					vm.fileList = [];
					vm.fileName = '';
					vm.$refs.gsmLicenseForm.resetFields();
					vm.gsmImportModeSelection = 'moreFile';
					vm.gsmFileData =[];
				}else if(vm.commonImportType == 'batchParams'){
					vm.fileList = [];
					vm.fileName = '';
					vm.importRuleForm = {
						operType: "0",
						filePath: ""
					}
					vm.$refs.gsmImportRuleForm.resetFields();
					vm.gsmCommonImportShow = false;
				}
			},	
			//query
			queryGsmLicense(text) {
				var vm = this;
				vm.gsmLicenseQuery.searchText = text;
				vm.gsmLicenseQuery.productType = vm.gsmAddOrEditForm.productType;
			},
			//status format
			executeStatus(row,column,value,rowIndex){
				if ('0' == value) {
					return '<%=rb.getString("WeiZhiXing")%>';
				} else if ('1' == value){
					return '<%=rb.getString("ZhengZaiZhiXing")%>';
				} else if ('2' == value){
					return '<%=rb.getString("ZhiXingChengGong")%>';
				} else if ('3' == value){
					return '<%=rb.getString("ZhiXingShiBai")%>';
				}else{
					return '';
				}
			},
			//------------------------------------------------------------parameter config------------------------------------------------------------
			gsmClickTab(val){
        		var vm = this;
				if(vm.gsmParameterConfigActive == 'gsmParamsImport'){
					//vm.gsmUrlConfigPlan = '${ctx}/gnb/pnp/queryBasicConfigPageList.action'; ////???记得改这里啊
					vm.gsmUrlConfigPlan = '${ctx}/gsm/pnp/queryBatchBasicConfigPageList.action';  
				}
        		vm.gsmCommonRightModelClose();
			},
			commonAddOrUpdateParamClick(row, operType, operModel, propsRow){//新建，修改按钮点击：行数据，操作类型，操作模块   ------------------------------- advance 
				var vm = this;
				vm.gsmCommonAdvanceOperType = operType;
				vm.gsmCommonAdvanceOperModel = operModel;
				if(operType == 'add'){
					vm.gsmCommonAddOrUpdateParamTitle = '<%=rb.getString("TianJia")%>';
					vm.$refs.gsmCommonAddEditForm.resetFields();
				}else{
					vm.gsmCommonAddOrUpdateParamTitle = '<%=rb.getString("XiuGai")%>';
				}
				// Static Routing List
				if(operModel == 'router'){
					if(operType == 'edit'){
						Object.assign(vm.gsmCommonAddEditForm, row) 
					}else{
						vm.gsmCommonAddEditForm.onboot = '1';
						vm.gsmCommonAddEditForm.gateway = '';
						vm.gsmCommonAddEditForm.netAddr = '';
						vm.gsmCommonAddEditForm.netMask = '';
					}
				}
				if(operModel == 'wan'){
					if(operType == 'edit'){
						Object.assign(vm.gsmCommonAddEditForm, row);
					}else{
						vm.gsmCommonAddEditForm.enable = '1';
						vm.gsmCommonAddEditForm.ipMode = '0';
						vm.gsmCommonAddEditForm.ipAddr = '';
						vm.gsmCommonAddEditForm.netMask = '';
						vm.gsmCommonAddEditForm.gateway = '';
						vm.gsmCommonAddEditForm.vlanId = '';
					}
				}
				vm.gsmCommonAddOrUpdateParamShow = true; //参数配置 batch import edit
				vm.gsmAddVersionShow = false; //版本窗口
				vm.gsmCommonImportShow = false; //license or batch import
				vm.gsmCommonImportFileInfoShow = false; //batch import view
			},
			commonDelParamClick(row, operModel, propsRow){//advance: delete
				var vm = this, confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					if(operModel == 'router'){//advance / Static Routing List: delete
						vm.gsmAddOrEditForm.staticRoute.map(function(item,index){
							if(item.id == row.id){
								vm.gsmAddOrEditForm.staticRoute = vm.gsmAddOrEditForm.staticRoute.filter((items)=>{
									return items.id != row.id
								})
							}
						})
					}else if(operModel == 'wan'){
						vm.gsmAddOrEditForm.wan.map(function(item,index){
							if(item.id == row.id){
								vm.gsmAddOrEditForm.wan = vm.gsmAddOrEditForm.wan.filter((items)=>{
									return items.id != row.id
								})
							}
						})
					}
				})
			},
			commonAddOrUpdateParamSubmit(){
				var vm = this, params = {}, id = 'id';
				vm.$refs.gsmCommonAddEditForm.validate(function(valid){
					if(valid){
						if(vm.gsmCommonAdvanceOperModel == 'router'){
							params= {
								onboot: vm.gsmCommonAddEditForm.onboot,
								gateway: vm.gsmCommonAddEditForm.gateway,
								netAddr: vm.gsmCommonAddEditForm.netAddr,
								netMask: vm.gsmCommonAddEditForm.netMask,
								id: vm.gsmCommonAddEditForm.id ? vm.gsmCommonAddEditForm.id : ''
							};
							if(vm.gsmCommonAdvanceOperType == 'add'){
								if(vm.gsmAddOrEditForm.staticRoute.length == 0){
									params[id] = '1';
								}else{
									var idList=[];
									vm.gsmAddOrEditForm.staticRoute.map((item)=>{
										idList.push(item[id] + '');
									})
									params[id] = vm.createId(1,idList); 
								}
								vm.gsmAddOrEditForm.staticRoute.push(params);
							}else{
								var idx='';
								vm.gsmAddOrEditForm.staticRoute.map((item,index)=>{
									if(item[id] == params[id]){
										idx = index
									}
								})
								Object.assign(vm.gsmAddOrEditForm.staticRoute[idx],params)
							}
						}
						if(vm.gsmCommonAdvanceOperModel == 'wan'){
							params= {
								enable: vm.gsmCommonAddEditForm.enable,
								ipMode: vm.gsmCommonAddEditForm.ipMode,
								ipAddr: vm.gsmCommonAddEditForm.ipAddr,
								netMask: vm.gsmCommonAddEditForm.netMask,
								gateway: vm.gsmCommonAddEditForm.gateway,
								vlanId: vm.gsmCommonAddEditForm.vlanId,
								id: vm.gsmCommonAddEditForm.id ? vm.gsmCommonAddEditForm.id : ''
							};
							
							if(vm.gsmCommonAdvanceOperType == 'add'){
								if(vm.gsmAddOrEditForm.wan.length == 0){
									params[id] = '1';
								}else{
									var idList=[];
									vm.gsmAddOrEditForm.wan.map((item)=>{
										idList.push(item[id] + '');
									})
									params[id] = vm.createId(1,idList); 
								}
								vm.gsmAddOrEditForm.wan.push(params);
							}else{
								var idx='';
								vm.gsmAddOrEditForm.wan.map((item,index)=>{
									if(item[id] == params[id]){
										idx = index
									}
								})
								Object.assign(vm.gsmAddOrEditForm.wan[idx],params)
							}
						}
						
						vm.commonAddOrUpdateParamCancel();
					}
				});
			},
			//advance: 关闭右侧窗口
			commonAddOrUpdateParamCancel(){
				var vm = this;
				vm.gsmCommonAddOrUpdateParamShow = false;
			},
			addNrPlmnAddClick(){// 修改NR CELL弹窗中 plmnList 新增
				var vm = this, 
					id = 'id', 
					params = { custParamName: '', custParamValue: '', custParamPath: ''};
				if(vm.gsmAddOrEditForm.custParam.length == 0){
					params[id] = '1';
				}else{
					var idList=[];
					vm.gsmAddOrEditForm.custParam.map((item)=>{
						idList.push(item.id + '');
					})
					params.id = vm.createId(1,idList); 
					var delNum = 0;
					vm.gsmAddOrEditForm.custParam.map((item)=>{
						delNum+=1;
					})
				}
				vm.gsmAddOrEditForm.custParam.push(params);
				event.stopPropagation();
			},
			addNrPlmnListDel(row){//Customized Paeameters : delete
				var vm = this;
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					vm.gsmAddOrEditForm.custParam.map(function(item,index){
						if(item.id == row.id){
							vm.gsmAddOrEditForm.custParam = vm.gsmAddOrEditForm.custParam.filter((items)=>{
								return items.id != row.id
							})
						}
					})
				}).catch(()=>{})
			},
			
			//batch import icon click
			gsmParamImportFileClick(){
				var vm = this;
				vm.commonImportTitle = '<%=rb.getString("DaoRu")%>';
				vm.commonImportType = 'batchParams';
				vm.gsmCommonImportShow = true;
				vm.commonImportFileLoading = false;
				vm.gsmCommonUploadBtnDisabled = false;
				vm.gsmAddVersionShow = false; 
				vm.gsmCommonAddOrUpdateParamShow = false; //batch import edit
				vm.gsmCommonImportFileInfoShow = false; //batch import view
				vm.$refs.gsmImportRuleForm.resetFields();
			},
			//check file
			gsmCheckFile(res,file){ 
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
					vm.$refs.paramsImportFileTable.refresh();
					vm.gsmCloseFileSelect();
					vm.gsmCommonImportShow = false;
				}else{
					vm.$message({
						type: 'error',
						message: res.msg
					});
				}
				var fileList = vm.$refs.upload.uploadFiles;
				fileList.forEach(function(file){
					file.status = 'ready';
				})
			},
			gsmFileChange(file,fileList){ 
				var vm = this;
				vm.fileName = file.name;
				vm.fileParams.FileName = file.name;
			},
			gsmFileSelect(){  
				var vm =this;
				vm.$refs.upload.clearFiles();
				vm.$refs['file_up'].click();
			},
			gsmCloseFileSelect(){
				var vm = this;
				vm.fileName = '';			
				vm.$refs.upload.clearFiles();
			},
			gsmBeforeUpload(file){
				var vm = this, 
					fileName = file.name,
					fileSize = file.size, 
					commonId = '', 
					fd = new FormData(),
					config = {
						headers: { 'Content-Type': 'multipart/form-data' }
					};
				commonId = vm.gsmOperType == 'add' ? '${policyId}' : vm.gsmPolicyId;
				fd.append('operType',vm.gsmImportRuleForm.operType);
				fd.append('policyId', commonId);
				fd.append('uploadFile',file); 
				vm.commonImportFileLoading = true;
				axios.post('${ctx}/gsm/pnp/importBatchConfigInfos.action',fd,config).then(function(res){
					var data = res.data;
					if(data["success"]){	
						vm.$message.success('<%=rb.getString("ChengGong")%>');
						vm.$refs.paramsImportFileTable.refresh();
						vm.gsmCommonImportCancel();
					}else{
						vm.gsmCommonImportCancel();
						if(data["msg"] == "1"){
							vm.$message.error('<%=rb.getString("DaoRuShiBai")%>');
						}else if(data["msg"] == "2"){
							vm.$refs.paramsImportFileTable.refresh();
							vm.downloadVisible = true;
					   }else{
						   vm.$message.error('<%=rb.getString("DaoRuShiBai")%>')
					   }
					}
					vm.commonImportFileLoading = false;
					vm.gsmCommonUploadBtnDisabled = false;
				})
				return false;
			},
			//batch import: close
			closeDownloadDialog(){
				var vm = this;
				vm.downloadVisible = false;
			},
			//batch import: download error file
			downloadErrorFile(){
				var vm = this;
				exportByForm('${ctx}/gsm/pnp/downloadFailureFile.action',{});
				vm.downloadVisible = false;
			},		
			//batch import download template
			gsmExportTemplate(){
				exportByForm('${ctx}/gsm/pnp/downloadSelfConfigTemp.action', {});
			},
			//batch import list export
			gsmParamExportClick(){
				/*var vm = this, 
					params = {search_text: this.importParamsQuery.searchText, timeZone : timeZone}
				if(vm.gsmOperType == 'modify'){
					params.policyId = vm.curPolicyId;
				}
				exportByForm('',params);*/
			},
			//params config import table edit
			paramsImportTableEditClick(row, evt){//--------------------------------------------------------------- params config import
				var vm = this,
					commonParams = {};

				vm.commonImportRow = row;

				if(vm.gsmOperType == 'add'){
					commonParams.policyId = '${policyId}';
				}else{
					commonParams.policyId = vm.gsmPolicyId;
				}
				commonParams.serialNumber = row.serialNumber;  // 批量配置传递 sn, 公共配置传递default
				axios.post('${ctx}/gsm/pnp/getPnPPolicyConfigInfo.action', stringify(commonParams)).then(function(res){//查询所有参数接口
					var data = res.data;
					
					if(data){
						Object.assign(vm.gsmImportAddOrEditForm, data);							
					}					
				});

				vm.paramsImportEditDialogShow = true;
				vm.gsmAddVersionShow = false; //version 
				vm.gsmCommonImportShow = false; //license or batch import
				vm.gsmCommonAddOrUpdateParamShow = false; //batch import edit
				vm.gsmCommonImportFileInfoShow = false; //batch import view
			},
			paramsImportEditDialogSubmit(){//batch import: edit: 一级弹窗 提交
				var vm = this, params = {}, paramsKeyList = [];

				params.policyId = vm.gsmOperType == 'add' ? '${policyId}' : vm.gsmPolicyId;
				paramsKeyList = ['ipaUnitid','omlRemoteIp','omlRemoteIpBak','rfPower'];
				paramsKeyList.map(function(prop){
					params[prop]= vm.gsmImportAddOrEditForm[prop];	
				});

				params.serialNumber = vm.commonImportRow.serialNumber;				
				params.staticRoute = JSON.stringify(vm.gsmImportAddOrEditForm.staticRoute);
				params.wan = JSON.stringify(vm.gsmImportAddOrEditForm.wan);
				
				vm.$refs.gsmImportAddOrEditForm.validate(function(valid){
					if(valid){
						vm.gsmParamImportSaveBtnDisabled = true;
						
                        axios.post('${ctx}/gsm/pnp/addOrModifyBatchConfigInfo.action',stringify(params)).then(function(res){
                        	var data = res.data;
                        	if(data){
                        		vm.gsmParamImportSaveBtnDisabled = false;
                        		if(data['success'] == true) {
                                    vm.$message({
    		    						message: '<%=rb.getString("ChengGong")%>',
    		    						type:'success',
    		    					});
                                    vm.$refs.paramsImportFileTable.refresh();
                                    vm.paramsImportEditDialogClose();
                                }else {
                                    vm.$message.error(data['message']);
                                }
                        	}
                        }).catch(function(){});
					}
				})
			},
			paramsImportEditDialogClose(){//batch import: edit: 一级弹窗 关闭
				var vm = this;
				vm.$refs.gsmImportAddOrEditForm.resetFields();
				vm.paramsImportEditDialogShow = false;
			},
			importParamClick(row, operType, operModel, propsRow){//新建，修改按钮点击：行数据，操作类型(add,edit)，操作模块(route,wan) 行数据
				var vm = this;
				vm.gsmCommonImportOperType = operType;
				vm.gsmCommonImportOperModel = operModel;

				if(operModel == 'importRoute'){
					vm.gsmImportSaveForm.addressType = 'Static';
					if(operType == 'edit'){
						Object.assign(vm.gsmImportSaveForm, row) 
					}else{
						vm.gsmImportSaveForm.onboot = '1';
						vm.gsmImportSaveForm.gateway = '';
						vm.gsmImportSaveForm.netAddr = '';
						vm.gsmImportSaveForm.netMask = '';
					}
				}
				if(operModel == 'importWan'){
					if(operType == 'edit'){
						Object.assign(vm.gsmImportSaveForm, row) 
					}else{
						vm.gsmImportSaveForm.enable = '1';
						vm.gsmImportSaveForm.ipMode = '0';
						vm.gsmImportSaveForm.ipAddr = '';
						vm.gsmImportSaveForm.netMask = '';
						vm.gsmImportSaveForm.gateway = '';
						vm.gsmImportSaveForm.vlanId = '';
					}
				}

				vm.paramsImportSaveDialogShow = true; //参数配置 commonAddOrUpdateParam
			},
			importParamDelClick(row, operModel, propsRow){
				var vm = this, confirmStr = '<%=rb.getString("QueRenShanChu")%>';
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(()=>{
					if(operModel == 'importWan'){//advance / Static Routing List: delete
						vm.gsmImportAddOrEditForm.wan.map(function(item,index){
							if(item.id == row.id){
								vm.gsmImportAddOrEditForm.wan = vm.gsmImportAddOrEditForm.wan.filter((items)=>{
									return items.id != row.id
								})
							}
						})
					}else if(operModel == 'importRoute'){
						vm.gsmImportAddOrEditForm.staticRoute.map(function(item,index){
							if(item.id == row.id){
								vm.gsmImportAddOrEditForm.staticRoute = vm.gsmImportAddOrEditForm.staticRoute.filter((items)=>{
									return items.id != row.id
								})
							}
						})
					}
				})
			},
			paramsImportSaveDialogSubmit(){//batch import: edit: 二级弹窗 提交
				var vm = this, params = {}, id = 'id';

				vm.$refs.gsmImportSaveForm.validate(function(valid){
					if(valid ){
						if(vm.gsmCommonImportOperModel == 'importRoute'){
							params= {
								onboot: vm.gsmImportSaveForm.onboot,
								gateway: vm.gsmImportSaveForm.gateway,
								netAddr: vm.gsmImportSaveForm.netAddr,
								netMask: vm.gsmImportSaveForm.netMask,
								id: vm.gsmImportSaveForm.id ? vm.gsmImportSaveForm.id : '',
							};
							if(vm.gsmCommonImportOperType == 'add'){
								if(vm.gsmImportAddOrEditForm.staticRoute.length == 0){
									params[id] = '1';
								}else{
									var idList=[];
									vm.gsmImportAddOrEditForm.staticRoute.map((item)=>{
										idList.push(item[id] + '');
									})
									params[id] = vm.createId(1,idList); 
								}
								vm.gsmImportAddOrEditForm.staticRoute.push(params);
							}else{
								var idx='';
								vm.gsmImportAddOrEditForm.staticRoute.map((item,index)=>{
									if(item[id] == params[id]){
										idx = index
									}
								})
								Object.assign(vm.gsmImportAddOrEditForm.staticRoute[idx],params)
							}							
						} 
						
						if(vm.gsmCommonImportOperModel == 'importWan'){
							params= {
								enable: vm.gsmImportSaveForm.enable,
								ipMode: vm.gsmImportSaveForm.ipMode,
								ipAddr: vm.gsmImportSaveForm.ipAddr,
								netMask: vm.gsmImportSaveForm.netMask,
								gateway: vm.gsmImportSaveForm.gateway,
								vlanId: vm.gsmImportSaveForm.vlanId,
								id: vm.gsmImportSaveForm.id ? vm.gsmImportSaveForm.id : ''
							};
							
							if(vm.gsmCommonImportOperType == 'add'){
								if(vm.gsmImportAddOrEditForm.wan.length == 0){
									params[id] = '1';
								}else{
									var idList=[];
									vm.gsmImportAddOrEditForm.wan.map((item)=>{
										idList.push(item[id] + '');
									})
									params[id] = vm.createId(1,idList); 
								}
								vm.gsmImportAddOrEditForm.wan.push(params);
							}else{
								var idx='';
								vm.gsmImportAddOrEditForm.wan.map((item,index)=>{
									if(item[id] == params[id]){
										idx = index
									}
								})
								Object.assign(vm.gsmImportAddOrEditForm.wan[idx],params)
							}
						}
						
						vm.paramsImportSaveDialogClose();
					}
				});
			},
			paramsImportSaveDialogClose (){//batch import: edit: 二级弹窗 关闭
				var vm = this;
				vm.$refs.gsmImportSaveForm.resetFields();
				vm.paramsImportSaveDialogShow = false;
			},
			//--------------------info
			paramsImportTableViewClick(row, evt){
				var vm = this, commonParams = {}; 

				vm.importFileInfoSN = row.serialNumber;
				if(vm.gsmOperType == 'add'){
					commonParams.policyId = '${policyId}';
				}else{
					commonParams.policyId = vm.gsmPolicyId;
				}
				commonParams.serialNumber = row.serialNumber;

				vm.gsmCommonImportFileInfoShow = true;//batch import view
				vm.gsmCommonImportShow = false;//license or batch import
				vm.gsmAddVersionShow = false; //version
				vm.gsmCommonAddOrUpdateParamShow = false; //参数配置 commonAddOrUpdateParam

				axios.post('${ctx}/gsm/pnp/getPnPPolicyConfigInfo.action', stringify(commonParams)).then(function(res){//查询所有参数接口
					var data = res.data;
					
					if(data){
						Object.assign(vm.gsmParamsImportInfo, data);	
					}					
				});
			},
			commonImportFileInfoCancel(){
				var vm = this;
				vm.gsmCommonImportFileInfoShow = false;
			},
			// -------------------delete 
			paramsImportTableDelClick(row, evt){
				var vm = this,
					params = {serialNumbers: row.serialNumber};
				if(vm.gsmOperType == 'add'){
					params.policyId = '${policyId}';
				}else{
					params.policyId = vm.gsmPolicyId;
				}
				vm.gsmCommonImportFileInfoShow = false;
				vm.$confirm('<%=rb.getString("QueRenShanChu")%>','<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					//????确认这个接口
					axios.post("${ctx}/gsm/pnp/delPnPPolicyConfigInfo.action",stringify(params)).then(function(response){
						var data = response.data;
						var message = '<%=rb.getString("ChengGong")%>';
						if(data["success"]){
							vm.$message({
	    						message:message,
	    						type:'success',
	    					})
                            vm.$refs.paramsImportFileTable.refresh();
						}else{
							vm.$message.error(data["message"])
						}
					})
				}).catch(() => {})

				vm.gsmAddVersionShow = false; //version 
				vm.gsmCommonImportShow = false; //license or batch import
				vm.gsmCommonAddOrUpdateParamShow = false; //batch import edit
				vm.gsmCommonImportFileInfoShow = false; //batch import view
			},

			//batch import list query
			gsmBatchImportQuery(text){
				var vm = this;
				vm.importParamsQuery.searchText = text;
			},

			//--------------------------------
			//add policy submit
			gsmAddOrUpdateSubmit(){
				var vm = this, 
					params = {}, 
					paramsDataList = [];   
				params.policyId = vm.gsmOperType == 'modify' ? vm.gsmPolicyId : '${policyId}';
				paramsDataList = ['policySwitch','policyName','productType','executeType','upgradeEnable','targetVersion','preserveSetting', 'licenseEnable', 'selfConfigEnable', 'paramConfigEnable', 'advanceEnable', 'batchConfigEnable', 'ipaUnitid', 'omlRemoteIp', 'omlRemoteIpBak', 'rfPower', 'dns1', 'dns2', 'localTimezoneName'];
				paramsDataList.map(function(prop){
					params[prop]= vm.gsmAddOrEditForm[prop];						
				});
                if(vm.specifyVersionType == '1'){
                	params.originalVersion = 'all';
                }else{
                	var resultParam = vm.gsmResultOriginalVersionList.map(function(item){ 
						return item.originalVersion;
					}).join(',');
					params.originalVersion = vm.gsmResultOriginalVersionList.length > 0 ? resultParam : '';
                }

				params.staticRoute = JSON.stringify(vm.gsmAddOrEditForm.staticRoute);
				params.wan = JSON.stringify(vm.gsmAddOrEditForm.wan);
				params.custParam = JSON.stringify(vm.gsmAddOrEditForm.custParam);
			
				vm.$refs.gsmAddOrEditForm.validate(function(valid){
					if(valid){
						vm.gsmSaveBtnDisabled = true;
						axios.post('${ctx}/gsm/pnp/addPnPPolicy.action',stringify(params)).then(function(response){
                        	var data = response.data;
                        	if(data){
                        		 vm.gsmSaveBtnDisabled = false;
                        		if(data['success'] == true) {
                                    vm.$message({
    		    						message: '<%=rb.getString("ChengGong")%>',
    		    						type:'success',
    		    					});
                                    eventBus.$emit('gsm_reload-config-list');
                                    vm.gsmAddOrUpdateCancel();
                                }else {
                                    vm.$message.error(data['message']);
                                }
                        	}
                        }).catch(function(){});
					}
				})
			},
			//slide close
			gsmAddOrUpdateCancel(){
				window.plugAndPlayVue.tabShow = 'itemHide';
				gsmPlugAndPlayVue.$refs.gsmSlider.hide();
			},

			createId(idVal,list){
				var vm = this, val = idVal + '';
				if(list.includes(val) == true){
					idVal += 1 ;
					return vm.createId(idVal,list);
				}else{
					return  idVal + '';
				}
			},
			isValidIP(ip){//校验IP
				var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
				return reg.test(ip);     
			},
			isIPv6(str){ //Ipv6校验 
				var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
				return reg.test(str);
			},
			isMask(str){ //校验子网掩码
				var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
				return exp.test(str); 		
			},
			isNumeric(str) {// 验证输入的是否是数字
				if(str.length==0){ return false; }
				for(var i=0;i<str.length;i++){
					if(str.charAt(i)<"0" || str.charAt(i)>"9"){
						return false;
					}
				}
				return true;  
			}
        },
        mounted() {
        	var vm = this;
			eventBus.$off('gsmInit-plugPlayConfig').$on('gsmInit-plugPlayConfig', vm.gsmPolicyInit);
            eventBus.$off('gsmSave-plugPlayConfig').$on('gsmSave-plugPlayConfig', vm.gsmAddOrUpdateSubmit);
            eventBus.$off('gsmHandle-plugPlayCancel').$on('gsmHandle-plugPlayCancel',vm.gsmAddOrUpdateCancel);
        }
    });
</script>